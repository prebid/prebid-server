package yieldmo

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type YieldmoAdapter struct {
	endpoint string
}

type Ext struct {
	PlacementId string `json:"placement_id"`
}

func (a *YieldmoAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	var errs []error
	var adapterRequests []*adapters.RequestData

	adapterReq, errors := a.makeRequest(request)
	if adapterReq != nil {
		adapterRequests = append(adapterRequests, adapterReq)
	}
	errs = append(errs, errors...)

	return adapterRequests, errors
}

func (a *YieldmoAdapter) makeRequest(request *openrtb.BidRequest) (*adapters.RequestData, []error) {
	var errs []error

	if err := preprocess(request); err != nil {
		errs = append(errs, err)
	}

	// Last Step
	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
	}, errs
}

// Mutate the request to get it ready to send to yieldmo.
func preprocess(request *openrtb.BidRequest) error {
	for i := 0; i < len(request.Imp); i++ {
		var imp = request.Imp[i]
		var bidderExt adapters.ExtImpBidder

		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		var yieldmoExt openrtb_ext.ExtImpYieldmo

		if err := json.Unmarshal(bidderExt.Bidder, &yieldmoExt); err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		var impExt Ext
		impExt.PlacementId = yieldmoExt.PlacementId

		impExtJSON, err := json.Marshal(impExt)
		if err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		request.Imp[i].Ext = impExtJSON
	}

	return nil
}

// MakeBids make the bids for the bid response.
func (a *YieldmoAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: "banner",
			})
		}
	}
	return bidResponse, nil

}

func NewYieldmoBidder(endpoint string) *YieldmoAdapter {
	return &YieldmoAdapter{
		endpoint: endpoint,
	}
}
