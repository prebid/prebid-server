package rtbhouse

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

// Builder builds a new instance of the RTBHouse adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &RTBHouseAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

// RTBHouseAdapter implements the Bidder interface.
type RTBHouseAdapter struct {
	endpoint string
}

// MakeRequests prepares the HTTP requests which should be made to fetch bids.
func (adapter *RTBHouseAdapter) MakeRequests(
	openRTBRequest *openrtb2.BidRequest,
	reqInfo *adapters.ExtraRequestInfo,
) (
	requestsToBidder []*adapters.RequestData,
	errs []error,
) {
	openRTBRequestJSON, err := json.Marshal(openRTBRequest)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	requestToBidder := &adapters.RequestData{
		Method:  "POST",
		Uri:     adapter.endpoint,
		Body:    openRTBRequestJSON,
		Headers: headers,
	}
	requestsToBidder = append(requestsToBidder, requestToBidder)

	return requestsToBidder, errs
}

const unexpectedStatusCodeFormat = "" +
	"Unexpected status code: %d. Run with request.debug = 1 for more info"

// MakeBids unpacks the server's response into Bids.
func (adapter *RTBHouseAdapter) MakeBids(
	openRTBRequest *openrtb2.BidRequest,
	requestToBidder *adapters.RequestData,
	bidderRawResponse *adapters.ResponseData,
) (
	bidderResponse *adapters.BidderResponse,
	errs []error,
) {
	switch bidderRawResponse.StatusCode {
	case http.StatusOK:
		break
	case http.StatusNoContent:
		return nil, nil
	case http.StatusBadRequest:
		err := &errortypes.BadInput{
			Message: fmt.Sprintf(unexpectedStatusCodeFormat, bidderRawResponse.StatusCode),
		}
		return nil, []error{err}
	default:
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf(unexpectedStatusCodeFormat, bidderRawResponse.StatusCode),
		}
		return nil, []error{err}
	}

	var openRTBBidderResponse openrtb2.BidResponse
	if err := json.Unmarshal(bidderRawResponse.Body, &openRTBBidderResponse); err != nil {
		return nil, []error{err}
	}

	bidsCapacity := len(openRTBBidderResponse.SeatBid[0].Bid)
	bidderResponse = adapters.NewBidderResponseWithBidsCapacity(bidsCapacity)
	var typedBid *adapters.TypedBid
	for _, seatBid := range openRTBBidderResponse.SeatBid {
		for _, bid := range seatBid.Bid {
			bid := bid // pin! -> https://github.com/kyoh86/scopelint#whats-this
			typedBid = &adapters.TypedBid{Bid: &bid, BidType: "banner"}
			bidderResponse.Bids = append(bidderResponse.Bids, typedBid)
		}
	}

	return bidderResponse, nil

}
