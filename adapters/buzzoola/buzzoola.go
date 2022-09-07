package buzzoola

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint          string
	allowedCurrencies map[string]bool
}

type bidExt struct {
	BidType openrtb_ext.BidType `json:"bidType"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint:          config.Endpoint,
		allowedCurrencies: map[string]bool{"EUR": true, "RUB": true, "USD": true},
	}

	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	for i := range request.Imp {
		if request.Imp[i].BidFloorCur == "" {
			request.Imp[i].BidFloorCur = "USD"
		}

		if request.Imp[i].BidFloor > 0 && !a.allowedCurrencies[strings.ToUpper(request.Imp[i].BidFloorCur)] {
			convertedValue, err := requestInfo.ConvertCurrency(request.Imp[i].BidFloor, request.Imp[i].BidFloorCur, "RUB")
			if err != nil {
				return nil, []error{err}
			}

			request.Imp[i].BidFloorCur = "RUB"
			request.Imp[i].BidFloor = convertedValue
		}
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint,
		Body:   requestJSON,
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}

		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}

		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur

	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := bidType(bid.Ext)
			if err != nil {
				return nil, []error{err}
			}

			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}

	return bidResponse, nil
}

func bidType(bidExtRaw json.RawMessage) (openrtb_ext.BidType, error) {
	var bidExtParsed bidExt

	if err := json.Unmarshal(bidExtRaw, &bidExtParsed); err != nil {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Received incorrect bid.Ext: %s.", err.Error()),
		}
	}

	return bidExtParsed.BidType, nil
}
