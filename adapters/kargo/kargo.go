package kargo

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

type KargoAdapter struct {
	URI string
}

// Builder builds a new instance of the Kargo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &KargoAdapter{
		URI: config.Endpoint, // base url of bidding server
	}
	return bidder, nil
}

// MakeRequests creates outgoing requests to the Kargo bidding server.
func (a *KargoAdapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.URI,
		Body:   requestJSON,
	}

	return []*adapters.RequestData{requestData}, nil
}

// MakeBids receives a bid response from the Kargo bidding server and creates bids for the publishers auction.
func (a *KargoAdapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: getMediaTypeForImp(bid.ImpID, request.Imp),
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

// getMediaTypeForImp checks the media type on the request Impression.
func getMediaTypeForImp(impID string, reqImps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range reqImps {
		if imp.ID == impID {
			switch {
			case imp.Banner != nil:
				return openrtb_ext.BidTypeBanner
			case imp.Native != nil:
				return openrtb_ext.BidTypeNative
			case imp.Video != nil:
				return openrtb_ext.BidTypeVideo
			}
		}
	}

	return openrtb_ext.BidTypeBanner
}
