package pwbid

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
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

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint,
		Body:   requestJSON,
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))

	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, typeerr := getMediaTypeForBid(request.Imp, bid)
			if typeerr != nil {
				return nil, []error{typeerr}
			}
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}

	return bidResponse, nil
}

func getMediaTypeForBid(impressions []openrtb2.Imp, bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	for _, impression := range impressions {
		if impression.ID == bid.ImpID {
			if impression.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}
			if impression.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			}
			if impression.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			}
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("The impression with ID %s is not present into the request", bid.ImpID),
	}
}
