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
func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unknown status code: %d.", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unknown status code: %d.", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: getMediaTypeForBid(sb.Bid[i].ImpID, internalRequest.Imp),
			})
		}
	}
	return bidResponse, nil
}

func getMediaTypeForBid(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo
			} else if imp.Audio != nil {
				return openrtb_ext.BidTypeAudio
			}
		}
	}
	return openrtb_ext.BidTypeBanner
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
