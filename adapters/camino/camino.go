package camino

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// The Camino endpoint bids and reports floors in USD only.
const caminoCurrency = "USD"

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Camino adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	requestCopy := *request
	requestCopy.Cur = []string{caminoCurrency}

	imps := make([]openrtb2.Imp, 0, len(request.Imp))
	var publisherID string
	for _, imp := range request.Imp {
		// imp is a copy of an element which the core framework already shallow
		// copied, so its non-pointer fields are safe to write.
		params, err := hoistBidderParams(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := convertFloorToUSD(&imp, reqInfo); err != nil {
			errs = append(errs, err)
			continue
		}
		if publisherID == "" {
			publisherID = params.PublisherID
		}
		imps = append(imps, imp)
	}
	if len(imps) == 0 {
		return nil, errs
	}
	requestCopy.Imp = imps

	setPublisherID(&requestCopy, publisherID)
	if err := setSession(&requestCopy, sessionID(request)); err != nil {
		return nil, append(errs, err)
	}

	body, err := jsonutil.Marshal(&requestCopy)
	if err != nil {
		return nil, append(errs, err)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(imps),
	}}, errs
}

// hoistBidderParams reads the Camino params out of imp.ext and rewrites imp.ext into
// the shape the Camino endpoint expects: placementId at the top level, next to
// whatever Prebid already put there. Every other key (gpid, tid, data, ...) is
// preserved untouched - the endpoint forwards them to demand partners, and tids in
// particular must never be rewritten by an adapter.
func hoistBidderParams(imp *openrtb2.Imp) (openrtb_ext.ImpExtCamino, error) {
	var params openrtb_ext.ImpExtCamino

	ext := map[string]json.RawMessage{}
	if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
		return params, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: invalid imp.ext: %s", imp.ID, err),
		}
	}
	bidder, ok := ext["bidder"]
	if !ok {
		return params, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: missing imp.ext.bidder", imp.ID),
		}
	}
	if err := jsonutil.Unmarshal(bidder, &params); err != nil {
		return params, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: invalid imp.ext.bidder: %s", imp.ID, err),
		}
	}

	delete(ext, "bidder")
	if params.PlacementID != "" {
		placementID, err := jsonutil.Marshal(params.PlacementID)
		if err != nil {
			return params, err
		}
		ext["placementId"] = placementID
	}

	rewritten, err := jsonutil.Marshal(ext)
	if err != nil {
		return params, err
	}
	imp.Ext = rewritten
	return params, nil
}

// convertFloorToUSD restates a foreign-currency floor in USD. The Camino endpoint
// reads imp.bidfloor as USD unconditionally, so forwarding e.g. a EUR floor
// unconverted would silently misprice the impression.
func convertFloorToUSD(imp *openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) error {
	if imp.BidFloor <= 0 || imp.BidFloorCur == "" || strings.EqualFold(imp.BidFloorCur, caminoCurrency) {
		return nil
	}
	converted, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, caminoCurrency)
	if err != nil {
		return err
	}
	imp.BidFloor = converted
	imp.BidFloorCur = caminoCurrency
	return nil
}

// setPublisherID stamps the Camino publisher id onto site.publisher, replacing
// any id already there. Camino resolves ads.txt authorization, schain, floors,
// price factors and demand allowlists from this field, and an id sent on the
// request is typically the host's or publisher's own account id, not Camino's.
// The Prebid.js adapter overwrites it the same way.
//
// request.Site and site.Publisher are pointers into shared memory; both are
// shallow copied before the write.
func setPublisherID(request *openrtb2.BidRequest, publisherID string) {
	if publisherID == "" || request.Site == nil {
		return
	}

	siteCopy := *request.Site
	publisherCopy := openrtb2.Publisher{}
	if siteCopy.Publisher != nil {
		publisherCopy = *siteCopy.Publisher
	}
	publisherCopy.ID = publisherID
	siteCopy.Publisher = &publisherCopy
	request.Site = &siteCopy
}

// sessionID identifies the page view a bid request belongs to. Camino ties a
// slot's sequence of renders to it, so it has to stay stable across the requests
// of one page view: source.tid is Prebid's page-level transaction id, and the
// request id is the last resort.
func sessionID(request *openrtb2.BidRequest) string {
	if request.Source != nil && request.Source.TID != "" {
		return request.Source.TID
	}
	return request.ID
}

// setSession adds ext.session, the shape the Camino endpoint reads page-view
// context from. It merges into request.ext rather than replacing it so any other
// extension travelling on the request survives.
func setSession(request *openrtb2.BidRequest, id string) error {
	ext := map[string]json.RawMessage{}
	if len(request.Ext) > 0 {
		if err := jsonutil.Unmarshal(request.Ext, &ext); err != nil {
			return err
		}
	}

	session, err := jsonutil.Marshal(map[string]string{"id": id})
	if err != nil {
		return err
	}
	ext["session"] = session

	merged, err := jsonutil.Marshal(ext)
	if err != nil {
		return err
	}
	request.Ext = merged
	return nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = caminoCurrency
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]

			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			resolveMacros(&bid)

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
				BidMeta: &openrtb_ext.ExtBidPrebidMeta{AdvertiserDomains: bid.ADomain},
			})
		}
	}
	return bidResponse, errs
}

// getMediaTypeForBid maps the bid's mtype to the Prebid media type. The Camino
// endpoint sets mtype on every bid it returns, including bids relayed from its
// demand partners, so a bid without a supported mtype is rejected rather than
// guessed from the impression, which is ambiguous for multi-format imps.
func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported mtype %d for bid %s on impression %s", bid.MType, bid.ID, bid.ImpID),
		}
	}
}

// resolveMacros substitutes the cleared price into the creative and the win
// notice. Camino leaves ${AUCTION_PRICE} literal so that whoever renders the bid
// reports the price it actually cleared at.
func resolveMacros(bid *openrtb2.Bid) {
	price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
	bid.AdM = strings.ReplaceAll(bid.AdM, "${AUCTION_PRICE}", price)
	bid.NURL = strings.ReplaceAll(bid.NURL, "${AUCTION_PRICE}", price)
}
