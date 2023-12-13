package consumable

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"net/http"
)

type adapter struct {
	endpoint string
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	headers := http.Header{
		"Content-Type": {"application/json"},
		"Accept":       {"application/json"},
	}
	bodyBytes, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	if request.Site != nil {
		requests := []*adapters.RequestData{
			{
				Method:  "POST",
				Uri:     "https://e.serverbid.com/sb/rtb",
				Body:    bodyBytes,
				Headers: headers,
			},
		}
		return requests, errs

	} else {
		_, consumableExt, err := extractExtensions(request.Imp[0])
		if err != nil {
			return nil, err
		}
		var placementId = consumableExt.PlacementId
		requests := []*adapters.RequestData{
			{
				Method:  "POST",
				Uri:     "https://e.serverbid.com/rtb/bid?s=" + placementId,
				Body:    bodyBytes,
				Headers: headers,
			},
		}
		return requests, errs
	}

}
func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unknown status code: %d.", responseData.StatusCode),
		}}
	}

	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unknown status code: %d.", responseData.StatusCode),
		}}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	var errors []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			var bidVideo *openrtb_ext.ExtBidPrebidVideo
			if bidType == openrtb_ext.BidTypeVideo {
				bidVideo = &openrtb_ext.ExtBidPrebidVideo{Duration: int(bid.Dur)}
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:      &seatBid.Bid[i],
				BidType:  bidType,
				BidVideo: bidVideo,
			})
		}
	}
	return bidResponse, nil
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err := json.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			return openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to parse impression \"%s\" mediatype", bid.ImpID),
	}
}

// Builder builds a new instance of the Consumable adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func extractExtensions(impression openrtb2.Imp) (*adapters.ExtImpBidder, *openrtb_ext.ExtImpConsumable, []error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(impression.Ext, &bidderExt); err != nil {
		return nil, nil, []error{&errortypes.BadInput{
			Message: err.Error(),
		}}
	}

	var consumableExt openrtb_ext.ExtImpConsumable
	if err := json.Unmarshal(bidderExt.Bidder, &consumableExt); err != nil {
		return nil, nil, []error{&errortypes.BadInput{
			Message: err.Error(),
		}}
	}

	return &bidderExt, &consumableExt, nil
}
