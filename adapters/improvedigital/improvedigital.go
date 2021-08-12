package improvedigital

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

type ImprovedigitalAdapter struct {
	endpoint string
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *ImprovedigitalAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors = make([]error, 0)

	reqJSON, err := json.Marshal(request)
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
func (a *ImprovedigitalAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	if len(bidResp.SeatBid) > 1 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected SeatBid! Must be only one but have: %d", len(bidResp.SeatBid)),
		}}
	}

	seatBid := bidResp.SeatBid[0]

	if len(seatBid.Bid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(seatBid.Bid))

	for i := range seatBid.Bid {
		bid := seatBid.Bid[i]

		bidType, err := getMediaTypeForImp(bid.ImpID, internalRequest.Imp)
		if err != nil {
			return nil, []error{err}
		}

		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: bidType,
		})
	}
	return bidResponse, nil

}

// Builder builds a new instance of the Improvedigital adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &ImprovedigitalAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}

			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			}

			if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			}

			return "", &errortypes.BadServerResponse{
				Message: fmt.Sprintf("Unknown impression type for ID: \"%s\"", impID),
			}
		}
	}

	// This shouldnt happen. Lets handle it just incase by returning an error.
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to find impression for ID: \"%s\"", impID),
	}
}
