package sellwild

import (
	"fmt"
	"net/http"
	"net/url"
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

// requestKey groups imps into outbound requests. The exchange authenticates a request with the
// token of its first imp, so imps are only grouped when they share both partner code and token.
type requestKey struct {
	partnerCode string
	token       string
}

// Builder builds a new instance of the Sellwild adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpoint, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	return &adapter{endpoint: endpoint}, nil
}

// MakeRequests sends one request per partner code and token, since the partner code is part of
// the endpoint path and the exchange authenticates the partner with the token found in
// imp[0].ext.bidder. Imp.ext is forwarded untouched.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var keys []requestKey
	groups := make(map[requestKey][]openrtb2.Imp)

	for _, imp := range request.Imp {
		impExt, err := parseImpExt(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		key := requestKey{partnerCode: impExt.PartnerCode, token: impExt.Token}
		if _, exists := groups[key]; !exists {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], imp)
	}

	if len(keys) == 0 {
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.6")

	requests := make([]*adapters.RequestData, 0, len(keys))
	for _, key := range keys {
		imps := groups[key]

		endpoint, err := a.buildEndpointURL(key.partnerCode)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requestCopy := *request
		requestCopy.Imp = imps

		body, err := jsonutil.Marshal(requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requests = append(requests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     endpoint,
			Body:    body,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(imps),
		})
	}

	return requests, errs
}

func (a *adapter) buildEndpointURL(partnerCode string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{PublisherID: url.PathEscape(partnerCode)}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func parseImpExt(imp openrtb2.Imp) (openrtb_ext.ExtImpSellwild, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb_ext.ExtImpSellwild{}, &errortypes.BadInput{
			Message: fmt.Sprintf("invalid imp.ext for imp %s: %v", imp.ID, err),
		}
	}

	var sellwildExt openrtb_ext.ExtImpSellwild
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &sellwildExt); err != nil {
		return openrtb_ext.ExtImpSellwild{}, &errortypes.BadInput{
			Message: fmt.Sprintf("invalid imp.ext.bidder for imp %s: %v", imp.ID, err),
		}
	}

	if strings.TrimSpace(sellwildExt.PartnerCode) == "" {
		return openrtb_ext.ExtImpSellwild{}, &errortypes.BadInput{
			Message: fmt.Sprintf("missing partnerCode for imp %s", imp.ID),
		}
	}
	if strings.TrimSpace(sellwildExt.Token) == "" {
		return openrtb_ext.ExtImpSellwild{}, &errortypes.BadInput{
			Message: fmt.Sprintf("missing token for imp %s", imp.ID),
		}
	}

	return sellwildExt, nil
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
		return nil, []error{&errortypes.BadServerResponse{Message: err.Error()}}
	}

	// Bids are resolved against the imps of the outbound request that received this response,
	// not the whole auction, so a bid can never be attributed to another request's imp.
	var outbound openrtb2.BidRequest
	if err := jsonutil.Unmarshal(requestData.Body, &outbound); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(outbound.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(seatBid.Bid[i], outbound.Imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

// getMediaTypeForBid resolves the bid type from bid.mtype. The exchange only sets mtype when the
// underlying demand did, so when it is absent the type is taken from the impression, provided
// that impression declared exactly one of banner, video or native. The bid's impid must belong
// to imps either way.
func getMediaTypeForBid(bid openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	imp, found := findImp(bid.ImpID, imps)
	if !found {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unable to find imp %s for bid %s", bid.ImpID, bid.ID),
		}
	}

	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case 0:
		return getMediaTypeFromImp(bid, imp)
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported bid.mtype %d for bid %s in imp %s", bid.MType, bid.ID, bid.ImpID),
		}
	}
}

func findImp(impID string, imps []openrtb2.Imp) (openrtb2.Imp, bool) {
	for i := range imps {
		if imps[i].ID == impID {
			return imps[i], true
		}
	}
	return openrtb2.Imp{}, false
}

func getMediaTypeFromImp(bid openrtb2.Bid, imp openrtb2.Imp) (openrtb_ext.BidType, error) {
	formats := 0
	var bidType openrtb_ext.BidType
	if imp.Banner != nil {
		formats++
		bidType = openrtb_ext.BidTypeBanner
	}
	if imp.Video != nil {
		formats++
		bidType = openrtb_ext.BidTypeVideo
	}
	if imp.Native != nil {
		formats++
		bidType = openrtb_ext.BidTypeNative
	}

	if formats == 1 {
		return bidType, nil
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("bid %s in imp %s is missing mtype and the media type cannot be inferred from the imp", bid.ID, bid.ImpID),
	}
}
