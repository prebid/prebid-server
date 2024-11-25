package smartx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpointURL string
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpointURL: config.Endpoint,
	}, nil
}

func (a *adapter) MakeRequests(openRTBRequest *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) (requestsToBidder []*adapters.RequestData, errs []error) {
	openRTBRequestJSON, err := json.Marshal(openRTBRequest)
	if err != nil {
		errs = append(errs, fmt.Errorf("marshal bidRequest: %w", err))
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("x-openrtb-version", "2.5")

	if openRTBRequest.Device != nil {
		if openRTBRequest.Device.UA != "" {
			headers.Set("User-Agent", openRTBRequest.Device.UA)
		}

		if openRTBRequest.Device.IP != "" {
			headers.Set("Forwarded", "for="+openRTBRequest.Device.IP)
			headers.Set("X-Forwarded-For", openRTBRequest.Device.IP)
		}
	}

	return append(requestsToBidder, &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpointURL,
		Body:    openRTBRequestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(openRTBRequest.Imp),
	}), nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	if len(response.SeatBid) == 0 {
		return nil, []error{errors.New("no bidders found in JSON response")}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	var errs []error

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: openrtb_ext.BidTypeVideo,
			})
		}
	}

	return bidResponse, errs
}
