package rtbhouse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	BidderCurrency string = "USD"
)

// RTBHouseAdapter implements the Bidder interface.
type RTBHouseAdapter struct {
	endpoint string
}

// Builder builds a new instance of the RTBHouse adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &RTBHouseAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests prepares the HTTP requests which should be made to fetch bids.
func (adapter *RTBHouseAdapter) MakeRequests(
	openRTBRequest *openrtb2.BidRequest,
	reqInfo *adapters.ExtraRequestInfo,
) (
	requestsToBidder []*adapters.RequestData,
	errs []error,
) {

	reqCopy := *openRTBRequest
	reqCopy.Imp = []openrtb2.Imp{}
	for _, imp := range openRTBRequest.Imp {
		var bidFloorCur = imp.BidFloorCur
		var bidFloor = imp.BidFloor
		if bidFloorCur == "" && bidFloor == 0 {
			rtbhouseExt, err := getImpressionExt(imp)
			if err != nil {
				return nil, []error{err}
			}
			if rtbhouseExt.BidFloor > 0 && len(reqCopy.Cur) > 0 {
				bidFloorCur = reqCopy.Cur[0]
				bidFloor = rtbhouseExt.BidFloor
			}
		}

		// Check if imp comes with bid floor amount defined in a foreign currency
		if bidFloor > 0 && bidFloorCur != "" && strings.ToUpper(bidFloorCur) != BidderCurrency {
			// Convert to US dollars
			convertedValue, err := reqInfo.ConvertCurrency(bidFloor, bidFloorCur, BidderCurrency)
			if err != nil {
				return nil, []error{err}
			}

			bidFloorCur = BidderCurrency
			bidFloor = convertedValue
		}

		if bidFloor > 0 && bidFloorCur == BidderCurrency {
			// Update after conversion. All imp elements inside request.Imp are shallow copies
			// therefore, their non-pointer values are not shared memory and are safe to modify.
			imp.BidFloorCur = bidFloorCur
			imp.BidFloor = bidFloor
		}

		// Set the CUR of bid to BIDDER_CURRENCY after converting all floors
		reqCopy.Cur = []string{BidderCurrency}
		reqCopy.Imp = append(reqCopy.Imp, imp)
	}

	openRTBRequestJSON, err := json.Marshal(reqCopy)
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

func getImpressionExt(imp openrtb2.Imp) (*openrtb_ext.ExtImpRTBHouse, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Bidder extension not provided or can't be unmarshalled",
		}
	}

	var rtbhouseExt openrtb_ext.ExtImpRTBHouse
	if err := json.Unmarshal(bidderExt.Bidder, &rtbhouseExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error while unmarshaling bidder extension",
		}
	}

	return &rtbhouseExt, nil
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

	bidderResponse.Currency = BidderCurrency

	return bidderResponse, nil

}
