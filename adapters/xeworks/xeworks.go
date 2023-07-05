package xeworks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
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
	if err := json.Unmarshal(imp.Ext, &impExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("Failed to deserialize bidder impression extension: %v", err),
		}
	}

	var xeworksExt openrtb_ext.ExtXeworks
	if err := json.Unmarshal(impExt.Bidder, &xeworksExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("Failed to deserialize Xeworks extension: %v", err),
		}
	}

	endpointParams := macros.EndpointTemplateParams{
		Host:     xeworksExt.Env,
		SourceId: xeworksExt.Pid,
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
			Message: "Bidder Xeworks is unavailable. Please contact the bidder support.",
		}}
	}

	if err := adapters.CheckResponseStatusCodeForErrors(bidderRawResponse); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(bidderRawResponse.Body, &bidResp); err != nil {
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
			var bidExt bidExt
			if err := json.Unmarshal(bid.Ext, &bidExt); err != nil {
				errs = append(errs, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Failed to parse Bid[%d].Ext: %s", bidId, err.Error()),
				})
				continue
			}

			bidType, err := openrtb_ext.ParseBidType(bidExt.Prebid.Type)
			if err != nil {
				errs = append(errs, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Bid[%d].Ext.Prebid.Type expects one of the following values: 'banner', 'native', 'video', 'audio', got '%s'", bidId, bidExt.Prebid.Type),
				})
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
