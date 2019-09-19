package lockerdome

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const unexpectedStatusCodeMessage = "Unexpected status code: %d. Run with request.debug = 1 for more info"

// Implements Bidder interface.
type LockerDomeAdapter struct {
	endpoint string
}

// MakeRequests makes the HTTP requests which should be made to fetch bids [from the bidder, in this case, LockerDome]
func (adapter *LockerDomeAdapter) MakeRequests(openRTBRequest *openrtb.BidRequest, extraReqInfo *adapters.ExtraRequestInfo) (requestsToBidder []*adapters.RequestData, errs []error) {

	numberOfImps := len(openRTBRequest.Imp)

	if openRTBRequest.Imp == nil || numberOfImps == 0 { // lockerdometest/supplemental/empty_imps.json
		err := &errortypes.BadInput{
			Message: "No valid impressions in the bid request.",
		}
		errs = append(errs, err)
		return nil, errs
	}

	for i := 0; i < numberOfImps; i++ {
		// LockerDome currently only supports banner impressions, and requires data in the ext field.
		if openRTBRequest.Imp[i].Banner == nil { // lockerdometest/supplemental/unsupported_imp_type.json
			err := &errortypes.BadInput{
				Message: "LockerDome does not currently support non-banner types.",
			}
			errs = append(errs, err)
			return nil, errs
		}
		var bidderExt adapters.ExtImpBidder
		err := json.Unmarshal(openRTBRequest.Imp[i].Ext, &bidderExt)
		if err != nil { // lockerdometest/supplemental/no_ext.json
			err = &errortypes.BadInput{
				Message: "ext was not provided.",
			}
			errs = append(errs, err)
			return nil, errs
		}
		var lockerdomeExt openrtb_ext.ExtImpLockerDome
		err = json.Unmarshal(bidderExt.Bidder, &lockerdomeExt)
		if err != nil { // lockerdometest/supplemental/no_adUnitId_param.json
			err = &errortypes.BadInput{
				Message: "ext.bidder.adUnitId was not provided.",
			}
			errs = append(errs, err)
			return nil, errs
		}
		if lockerdomeExt.AdUnitId == "" { // lockerdometest/supplemental/empty_adUnitId_param.json
			err := &errortypes.BadInput{
				Message: "ext.bidder.adUnitId is empty.",
			}
			errs = append(errs, err)
			return nil, errs
		}
	}

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

	return requestsToBidder, nil

}

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
		return nil, []error{
			fmt.Errorf("Error unmarshaling LockerDome bid response - %s", err.Error()),
		}
	}

	if len(openRTBBidderResponse.SeatBid) == 0 {
		return nil, nil
	}

	bidderResponse = adapters.NewBidderResponseWithBidsCapacity(len(openRTBBidderResponse.SeatBid[0].Bid))

	for _, seatBid := range openRTBBidderResponse.SeatBid {
		for i := range seatBid.Bid {
			typedBid := adapters.TypedBid{Bid: &seatBid.Bid[i], BidType: openrtb_ext.BidTypeBanner}
			bidderResponse.Bids = append(bidderResponse.Bids, &typedBid)
		}
	}
	return bidderResponse, nil
}

func NewLockerDomeBidder(endpoint string) *LockerDomeAdapter {
	return &LockerDomeAdapter{endpoint: endpoint}
}
