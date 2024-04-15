package concert

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"

	"github.com/prebid/prebid-server/v2/adapters"
)

type adapter struct {
	endpoint string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (adapter *adapter) MakeRequests(openRTBRequest *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) (requestsToBidder []*adapters.RequestData, errs []error) {
	jsonBody, err := json.Marshal(openRTBRequest)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	request := &adapters.RequestData{
		Method:  "POST",
		Uri:     adapter.endpoint,
		Body:    jsonBody,
		Headers: headers,
	}

	requestsToBidder = append(requestsToBidder, request)

	return requestsToBidder, errs
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	bidResponse := new(openrtb2.BidResponse)
	if err := json.Unmarshal(response.Body, bidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: %s", err),
		}}
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResponse.SeatBid {
		for i := range sb.Bid {
			bidType, err := getBidType(sb.Bid[i].ImpID, internalRequest.Imp)
			if err != nil {
				return nil, []error{err}
			}
			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidderResponse, nil
}

func getBidType(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}
			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			}
			if imp.Audio != nil {
				return openrtb_ext.BidTypeAudio, nil
			}
		}
	}
	return "", fmt.Errorf("Unknown impression type for ID %s", impID)
}
