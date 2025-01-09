package iqx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type bidType struct {
	Type string `json:"type"`
}

type bidExt struct {
	Prebid bidType `json:"prebid"`
}

type adapter struct {
	endpoint *template.Template
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	tmpl, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint URL template: %v", err)
	}

	bidder := &adapter{
		endpoint: tmpl,
	}

	return bidder, nil
}

func (a *adapter) buildEndpointFromRequest(imp *openrtb2.Imp) (string, error) {
	var impExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &impExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("Failed to deserialize bidder impression extension: %v", err),
		}
	}

	var iqzonexExt openrtb_ext.ExtIQX
	if err := jsonutil.Unmarshal(impExt.Bidder, &iqzonexExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("Failed to deserialize IQZonex extension: %v", err),
		}
	}

	endpointParams := macros.EndpointTemplateParams{
		Host:     iqzonexExt.Env,
		SourceId: iqzonexExt.Pid,
	}

	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errs []error

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	requestCopy := *request
	for _, imp := range request.Imp {
		requestCopy.Imp = []openrtb2.Imp{imp}

		endpoint, err := a.buildEndpointFromRequest(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requestJSON, err := json.Marshal(requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		request := &adapters.RequestData{
			Method:  http.MethodPost,
			Body:    requestJSON,
			Uri:     endpoint,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
		}

		requests = append(requests, request)
	}

	return requests, errs
}

func (a *adapter) MakeBids(openRTBRequest *openrtb2.BidRequest, requestToBidder *adapters.RequestData, bidderRawResponse *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(bidderRawResponse) {
		return nil, nil
	}

	if bidderRawResponse.StatusCode == http.StatusServiceUnavailable {
		return nil, []error{&errortypes.BadInput{
			Message: "Bidder IQZonex is unavailable. Please contact the bidder support.",
		}}
	}

	if err := adapters.CheckResponseStatusCodeForErrors(bidderRawResponse); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(bidderRawResponse.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Array SeatBid cannot be empty",
		}}
	}

	return prepareBidResponse(bidResp.SeatBid)
}

func prepareBidResponse(seats []openrtb2.SeatBid) (*adapters.BidderResponse, []error) {
	errs := []error{}
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(seats))

	for _, seatBid := range seats {
		for bidId, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[bidId],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("failed to parse bid mtype for impression id \"%s\"", bid.ImpID)
	}
}
