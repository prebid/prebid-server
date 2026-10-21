package intenze

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct{}

var exchangeSubdomainsRegions = map[string]string{
	"us-east": "lb-east",
	"eu":      "n2",
	"apac":    "lb-apac",
}
var sspSubdomainsRegions = map[string]string{
	"us-east": "us-east-ssp",
	"eu":      "eu-ssp",
	"apac":    "apac-ssp",
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{}
	return bidder, nil
}

func getHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")

	if request.Device != nil {
		if len(request.Device.UA) > 0 {
			headers.Add("User-Agent", request.Device.UA)
		}

		if len(request.Device.IPv6) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}

		if len(request.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}
	}

	return headers
}

func impError(impID string, err error) error {
	return &errortypes.BadInput{
		Message: fmt.Sprintf("imp %s: %s", impID, err.Error()),
	}
}

func (a *adapter) MakeRequests(openRTBRequest *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	requests := make([]*adapters.RequestData, 0, len(openRTBRequest.Imp))

	for _, imp := range openRTBRequest.Imp {
		impExt, err := getImpressionExt(&imp)
		if err != nil {
			errs = append(errs, impError(imp.ID, err))
			continue
		}

		endpoint, err := a.buildEndpointURL(impExt)
		if err != nil {
			errs = append(errs, impError(imp.ID, err))
			continue
		}

		cleanExt, err := stripBidderParams(imp.Ext)
		if err != nil {
			errs = append(errs, impError(imp.ID, err))
			continue
		}
		imp.Ext = cleanExt

		reqCopy := *openRTBRequest
		reqCopy.Imp = []openrtb2.Imp{imp}

		reqJSON, err := jsonutil.Marshal(reqCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requests = append(requests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     endpoint,
			Body:    reqJSON,
			Headers: getHeaders(&reqCopy),
			ImpIDs:  openrtb_ext.GetImpIDs(reqCopy.Imp),
		})
	}

	return requests, errs
}

func stripBidderParams(ext json.RawMessage) (json.RawMessage, error) {
	if len(ext) == 0 {
		return nil, nil
	}

	var fields map[string]json.RawMessage
	if err := jsonutil.Unmarshal(ext, &fields); err != nil {
		return nil, err
	}

	delete(fields, "bidder")
	if len(fields) == 0 {
		return nil, nil
	}

	return jsonutil.Marshal(fields)
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtIntenze, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	var intenzeExt openrtb_ext.ExtIntenze
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &intenzeExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	return &intenzeExt, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtIntenze) (string, error) {
	if params.AccountID == nil && params.PlacementID == nil {
		return "", &errortypes.BadInput{
			Message: "either accountId or placementId expected",
		}
	}

	if params.AccountID != nil && params.PlacementID != nil {
		return "", &errortypes.BadInput{
			Message: "accountId and placementId cannot be used together",
		}
	}

	var route = ""
	var subdomain, queryKey, queryValue string
	var regionMap map[string]string

	if params.AccountID != nil {
		subdomain = "lb-east"
		queryKey = "pass"
		queryValue = *params.AccountID
		regionMap = exchangeSubdomainsRegions
	} else {
		subdomain = "us-east-ssp"
		queryKey = "placementId"
		route = "pbs-serve"
		queryValue = *params.PlacementID
		regionMap = sspSubdomainsRegions
	}

	if params.Region != nil {
		var exists bool
		subdomain, exists = regionMap[*params.Region]
		if !exists {
			return "", &errortypes.BadInput{
				Message: fmt.Sprintf("invalid region: %s. Expected us-east, eu or apac", *params.Region),
			}
		}
	}

	endpoint := fmt.Sprintf("https://%s.intenze.co/%s?%s=%s", subdomain, route, queryKey, url.QueryEscape(queryValue))

	return endpoint, nil
}

func checkResponseStatusCodes(response *adapters.ResponseData) error {
	if response.StatusCode == http.StatusServiceUnavailable {
		return &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Something went wrong Status Code: [ %d ] ", response.StatusCode),
		}
	}

	return adapters.CheckResponseStatusCodeForErrors(response)
}

func (a *adapter) MakeBids(openRTBRequest *openrtb2.BidRequest, requestToBidder *adapters.RequestData, bidderRawResponse *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(bidderRawResponse) {
		return nil, nil
	}

	httpStatusError := checkResponseStatusCodes(bidderRawResponse)
	if httpStatusError != nil {
		return nil, []error{httpStatusError}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(bidderRawResponse.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad Server Response",
		}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Empty SeatBid array",
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(requestToBidder.ImpIDs))
	if len(bidResp.Cur) != 0 {
		bidResponse.Currency = bidResp.Cur
	}

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for idx := range sb.Bid {
			bidType, err := getMediaTypeForImp(sb.Bid[idx])
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[idx],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

func getMediaTypeForImp(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported MType %d", bid.MType),
		}
	}
}
