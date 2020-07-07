package kubient

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
)

func NewKubientBidder(endpoint string) *KubientAdapter {
	return &KubientAdapter{endpoint: endpoint}
}

// Implements Bidder interface.
type KubientAdapter struct {
	endpoint string
}

// MakeRequests prepares the HTTP requests which should be made to fetch bids.
func (adapter *KubientAdapter) MakeRequests(
	openRTBRequest *openrtb.BidRequest,
	reqInfo *adapters.ExtraRequestInfo,
) ([]*adapters.RequestData, []error) {
	if len(openRTBRequest.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impression in the bid request",
		}}
	}
	errs := make([]error, 0, len(openRTBRequest.Imp))
	hasErrors := false
	for _, impObj := range openRTBRequest.Imp {
		err := checkImpExt(impObj)
		if err != nil {
			errs = append(errs, err)
			hasErrors = true
		}
	}
	if hasErrors {
		return nil, errs
	}
	openRTBRequestJSON, err := json.Marshal(openRTBRequest)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	requestsToBidder := []*adapters.RequestData{{
		Method:  "POST",
		Uri:     adapter.endpoint,
		Body:    openRTBRequestJSON,
		Headers: headers,
	}}
	return requestsToBidder, errs
}

func checkImpExt(impObj openrtb.Imp) error {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(impObj.Ext, &bidderExt); err != nil {
		return &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
	}
	var kubientExt openrtb_ext.ExtImpKubient
	if err := json.Unmarshal(bidderExt.Bidder, &kubientExt); err != nil {
		return &errortypes.BadInput{
			Message: "ext.bidder.zoneid is not provided",
		}
	}
	if kubientExt.ZoneID == "" {
		return &errortypes.BadInput{
			Message: "zoneid is empty",
		}
	}
	return nil
}

// MakeBids makes the bids
func (adapter *KubientAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

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

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType, err := getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp)
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					Bid:     &sb.Bid[i],
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
	return bidResponse, errs
}

func getMediaTypeForImp(impID string, imps []openrtb.Imp) (openrtb_ext.BidType, error) {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner == nil && imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			return mediaType, nil
		}
	}

	// This shouldnt happen. Lets handle it just incase by returning an error.
	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find impression \"%s\" ", impID),
	}
}
