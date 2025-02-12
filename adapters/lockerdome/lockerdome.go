package lockerdome

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const unexpectedStatusCodeMessage = "Unexpected status code: %d. Run with request.debug = 1 for more info"

// Implements Bidder interface.
type LockerDomeAdapter struct {
	endpoint string
}

// MakeRequests makes the HTTP requests which should be made to fetch bids [from the bidder, in this case, LockerDome]
func (adapter *LockerDomeAdapter) MakeRequests(openRTBRequest *openrtb2.BidRequest, extraReqInfo *adapters.ExtraRequestInfo) (requestsToBidder []*adapters.RequestData, errs []error) {

	numberOfImps := len(openRTBRequest.Imp)

	if openRTBRequest.Imp == nil || numberOfImps == 0 { // lockerdometest/supplemental/empty_imps.json
		err := &errortypes.BadInput{
			Message: "No valid impressions in the bid request.",
		}
		errs = append(errs, err)
		return nil, errs
	}

	var indexesOfValidImps []int
	for i := 0; i < numberOfImps; i++ {
		// LockerDome currently only supports banner impressions, and requires data in the ext field.
		if openRTBRequest.Imp[i].Banner == nil { // lockerdometest/supplemental/unsupported_imp_type.json
			err := &errortypes.BadInput{
				Message: "LockerDome does not currently support non-banner types.",
			}
			errs = append(errs, err)
			continue
		}
		var bidderExt adapters.ExtImpBidder
		err := jsonutil.Unmarshal(openRTBRequest.Imp[i].Ext, &bidderExt)
		if err != nil { // lockerdometest/supplemental/no_ext.json
			err = &errortypes.BadInput{
				Message: "ext was not provided.",
			}
			errs = append(errs, err)
			continue
		}
		var lockerdomeExt openrtb_ext.ExtImpLockerDome
		err = jsonutil.Unmarshal(bidderExt.Bidder, &lockerdomeExt)
		if err != nil { // lockerdometest/supplemental/no_adUnitId_param.json
			err = &errortypes.BadInput{
				Message: "ext.bidder.adUnitId was not provided.",
			}
			errs = append(errs, err)
			continue
		}
		if lockerdomeExt.AdUnitId == "" { // lockerdometest/supplemental/empty_adUnitId_param.json
			err := &errortypes.BadInput{
				Message: "ext.bidder.adUnitId is empty.",
			}
			errs = append(errs, err)
			continue
		}
		indexesOfValidImps = append(indexesOfValidImps, i)
	}
	if numberOfImps > len(indexesOfValidImps) {
		var validImps []openrtb2.Imp
		for j := 0; j < len(indexesOfValidImps); j++ {
			validImps = append(validImps, openRTBRequest.Imp[j])
		}
		if len(validImps) == 0 {
			err := &errortypes.BadInput{
				Message: "No valid or supported impressions in the bid request.",
			}
			errs = append(errs, err)
			return nil, errs
		} else {
			openRTBRequest.Imp = validImps
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
		ImpIDs:  openrtb_ext.GetImpIDs(openRTBRequest.Imp),
	}

	requestsToBidder = append(requestsToBidder, requestToBidder)

	return requestsToBidder, nil

}

// MakeBids unpacks the server's response into Bids.
func (adapter *LockerDomeAdapter) MakeBids(openRTBRequest *openrtb2.BidRequest, requestToBidder *adapters.RequestData, bidderRawResponse *adapters.ResponseData) (bidderResponse *adapters.BidderResponse, errs []error) {

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

	var openRTBBidderResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(bidderRawResponse.Body, &openRTBBidderResponse); err != nil {
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

// Builder builds a new instance of the LockerDome adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &LockerDomeAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
