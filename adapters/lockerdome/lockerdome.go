package lockerdome

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
)

// Implements Bidder interface.
type LockerDomeAdapter struct {
	endpoint string
}

// MakeRequests makes the HTTP requests which should be made to fetch bids [from the bidder, in this case, LockerDome]
func (adapter *LockerDomeAdapter) MakeRequests(openRTBRequest *openrtb.BidRequest, extraReqInfo *adapters.ExtraRequestInfo) (requestsToBidder []*adapters.RequestData, errs []error) {

	openRTBRequestJSON, err := json.Marshal(openRTBRequest)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	requestToBidder := &adapters.RequestData{
		Method:  "POST",
		Uri:     adapter.endpoint,
		Body:    openRTBRequestJSON,
		Headers: headers,
	}

	requestsToBidder = append(requestsToBidder, requestToBidder)

	return requestsToBidder, errs
}

const unexpectedStatusCodeMessage = "Unexpected status code: %d. Run with request.debug = 1 for more info"

// MakeBids unpacks the server's response into Bids.
func (adapter *LockerDomeAdapter) MakeBids(openRTBRequest *openrtb.BidRequest, requestToBidder *adapters.RequestData, bidderRawResponse *adapters.ResponseData) (bidderResponse *adapters.BidderResponse, errs []error) {

	if bidderRawResponse.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if bidderRawResponse.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf(unexpectedStatusCodeMessage, bidderRawResponse.StatusCode),
		}}
	}

	if bidderRawResponse.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf(unexpectedStatusCodeMessage, bidderRawResponse.StatusCode),
		}}
	}

	var openRTBBidderResponse openrtb.BidResponse
	if err := json.Unmarshal(bidderRawResponse.Body, &openRTBBidderResponse); err != nil {
		return nil, []error{err}
	}

	if len(openRTBBidderResponse.SeatBid) == 0 {
		return nil, nil
	}

	bidsCapacity := len(openRTBBidderResponse.SeatBid[0].Bid)
	bidderResponse = adapters.NewBidderResponseWithBidsCapacity(bidsCapacity)

	var typedBid adapters.TypedBid
	for _, seatBid := range openRTBBidderResponse.SeatBid {
		for i := range seatBid.Bid {
			typedBid = adapters.TypedBid{Bid: &seatBid.Bid[i], BidType: "banner"}
			bidderResponse.Bids = append(bidderResponse.Bids, &typedBid)
		}
	}
	return bidderResponse, nil
}

func NewLockerDomeBidder(endpoint string) *LockerDomeAdapter {
	return &LockerDomeAdapter{endpoint: endpoint}
}
