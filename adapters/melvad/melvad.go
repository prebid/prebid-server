package melvad

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/macros"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint *template.Template
}

// Builder builds a new instance of the Melvad adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpoint, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	return &adapter{endpoint: endpoint}, nil
}

// MakeRequests forwards the OpenRTB request as is. The impressions are grouped by sspId, because the sspId
// is a part of the endpoint URL: one request per SSP.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	impsBySsp := make(map[string][]openrtb2.Imp)
	var sspIDs []string

	for _, imp := range request.Imp {
		sspID, err := impSspID(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if _, exists := impsBySsp[sspID]; !exists {
			sspIDs = append(sspIDs, sspID)
		}

		impsBySsp[sspID] = append(impsBySsp[sspID], imp)
	}

	requests := make([]*adapters.RequestData, 0, len(sspIDs))

	for _, sspID := range sspIDs {
		requestData, err := a.makeRequest(request, sspID, impsBySsp[sspID])
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requests = append(requests, requestData)
	}

	return requests, errs
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest, sspID string, imps []openrtb2.Imp) (*adapters.RequestData, error) {
	uri, err := macros.ResolveMacros(a.endpoint, macros.EndpointTemplateParams{SspID: sspID})
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Unable to build the endpoint url: %v", err),
		}
	}

	requestCopy := *request
	requestCopy.Imp = imps

	body, err := jsonutil.Marshal(&requestCopy)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	// the device of an app or a site: its user agent and its IP, as a browser would send them
	if request.Device != nil {
		if request.Device.UA != "" {
			headers.Add("User-Agent", request.Device.UA)
		}

		if request.Device.IP != "" {
			headers.Add("X-Forwarded-For", request.Device.IP)
		} else if request.Device.IPv6 != "" {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}
	}

	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     uri,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(imps),
	}, nil
}

func impSspID(imp *openrtb2.Imp) (string, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("Impression %s: ext not valid or can't be unmarshalled: %v", imp.ID, err),
		}
	}

	var melvadExt openrtb_ext.ExtImpMelvad
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &melvadExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("Impression %s: ext.bidder not valid or can't be unmarshalled: %v", imp.ID, err),
		}
	}

	return melvadExt.SspID, nil
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
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: %v", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	var errs []error

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]

			bidType, err := getMediaTypeForBid(bid, request.Imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			resolveMacros(bid, bidResponse.Currency, request.ID)

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
				BidMeta: &openrtb_ext.ExtBidPrebidMeta{AdvertiserDomains: bid.ADomain},
			})
		}
	}

	return bidResponse, errs
}

// getMediaTypeForBid types a bid by its mtype, which the Melvad bidder sends with every bid, after checking that the
// bid answers an impression of the request that has that format. A bid without mtype takes the format of its
// impression only when the impression has one format; a guess between several formats is an error.
func getMediaTypeForBid(bid *openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	imp := impOf(bid.ImpID, imps)
	if imp == nil {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("The bid %s is for the unknown impression %s", bid.ID, bid.ImpID),
		}
	}

	switch {
	case bid.MType == openrtb2.MarkupBanner && imp.Banner != nil:
		return openrtb_ext.BidTypeBanner, nil
	case bid.MType == openrtb2.MarkupVideo && imp.Video != nil:
		return openrtb_ext.BidTypeVideo, nil
	case bid.MType == openrtb2.MarkupNative && imp.Native != nil:
		return openrtb_ext.BidTypeNative, nil
	case bid.MType == 0:
		if bidType, ok := onlyFormat(imp); ok {
			return bidType, nil
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Unable to determine the media type of the bid %s for the impression %s", bid.ID, bid.ImpID),
	}
}

// impOf returns the impression of the request with the id, nil when there is none.
func impOf(id string, imps []openrtb2.Imp) *openrtb2.Imp {
	for i := range imps {
		if imps[i].ID == id {
			return &imps[i]
		}
	}

	return nil
}

// onlyFormat returns the format of an impression that has exactly one of banner, video and native.
func onlyFormat(imp *openrtb2.Imp) (openrtb_ext.BidType, bool) {
	var formats []openrtb_ext.BidType
	if imp.Banner != nil {
		formats = append(formats, openrtb_ext.BidTypeBanner)
	}
	if imp.Video != nil {
		formats = append(formats, openrtb_ext.BidTypeVideo)
	}
	if imp.Native != nil {
		formats = append(formats, openrtb_ext.BidTypeNative)
	}

	if len(formats) != 1 {
		return "", false
	}

	return formats[0], true
}

// resolveMacros fills the auction macros of the markup and of the billing url: ${AUCTION_PRICE} with the bid price,
// ${AUCTION_CURRENCY} with the currency of the response and ${AUCTION_ID} with the id of the request.
func resolveMacros(bid *openrtb2.Bid, currency, auctionID string) {
	r := strings.NewReplacer(
		"${AUCTION_PRICE}", strconv.FormatFloat(bid.Price, 'f', -1, 64),
		"${AUCTION_CURRENCY}", currency,
		"${AUCTION_ID}", auctionID,
	)
	bid.AdM = r.Replace(bid.AdM)
	bid.BURL = r.Replace(bid.BURL)
}
