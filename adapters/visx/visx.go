package visx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type VisxAdapter struct {
	endpoint string
}

type visxBid struct {
	ImpID   string   `json:"impid"`
	Price   float64  `json:"price"`
	UID     int      `json:"auid"`
	CrID    string   `json:"crid,omitempty"`
	AdM     string   `json:"adm,omitempty"`
	ADomain []string `json:"adomain,omitempty"`
	DealID  string   `json:"dealid,omitempty"`
	W       uint64   `json:"w,omitempty"`
	H       uint64   `json:"h,omitempty"`
}

type visxSeatBid struct {
	Bid  []visxBid `json:"bid"`
	Seat string    `json:"seat,omitempty"`
}

type visxResponse struct {
	SeatBid []visxSeatBid `json:"seatbid,omitempty"`
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *VisxAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors = make([]error, 0)

	// copy the request, because we are going to mutate it
	requestCopy := *request
	if len(requestCopy.Cur) == 0 {
		requestCopy.Cur = []string{"USD"}
	}

	reqJSON, err := json.Marshal(requestCopy)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
	}}, errors
}

// MakeBids unpacks the server's response into Bids.
func (a *VisxAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var bidResp visxResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bid := openrtb2.Bid{}
			bid.ID = internalRequest.ID
			bid.CrID = sb.Bid[i].CrID
			bid.ImpID = sb.Bid[i].ImpID
			bid.Price = sb.Bid[i].Price
			bid.AdM = sb.Bid[i].AdM
			bid.W = int64(sb.Bid[i].W)
			bid.H = int64(sb.Bid[i].H)
			bid.ADomain = sb.Bid[i].ADomain
			bid.DealID = sb.Bid[i].DealID

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: "banner",
			})
		}
	}
	return bidResponse, nil

}

// Builder builds a new instance of the Visx adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &VisxAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
