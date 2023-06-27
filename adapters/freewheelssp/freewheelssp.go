package freewheelssp

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	for i := 0; i < len(request.Imp); i++ {
		imp := &request.Imp[i]
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Invalid imp.ext for impression index %d. Error Infomation: %s", i, err.Error()),
			}}
		}

		var impExt openrtb_ext.ImpExtFreewheelSSP
		if err := json.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Invalid imp.ext for impression index %d. Error Infomation: %s", i, err.Error()),
			}}
		}

		var err error
		if imp.Ext, err = json.Marshal(impExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Unable to transfer requestImpExt to Json fomat, %s", err.Error()),
			}}
		}
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unable to transfer request to Json fomat, %s", err.Error()),
		}}
	}

	headers := http.Header{}
	headers.Add("Componentid", "prebid-go")

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    requestJSON,
		Headers: headers,
	}
	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	cur := bidResp.Cur
	bidResponse := &adapters.BidderResponse{
		Currency: cur,
		Bids:     []*adapters.TypedBid{},
	}

	bidType := openrtb_ext.BidTypeVideo

	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		config.Endpoint,
	}
	return bidder, nil
}
