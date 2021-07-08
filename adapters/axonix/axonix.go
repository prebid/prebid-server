package axonix

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

type adapter struct {
	URI string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(request.Imp[0].Ext, &bidderExt); err != nil {
		errors = append(errors, &errortypes.BadInput{
			Message: err.Error(),
		})

		return nil, errors
	}

	var axonixExt openrtb_ext.ExtImpAxonix
	if err := json.Unmarshal(bidderExt.Bidder, &axonixExt); err != nil {
		errors = append(errors, &errortypes.BadInput{
			Message: err.Error(),
		})

		return nil, errors
	}

	thisURI := a.URI
	if len(thisURI) == 0 {
		thisURI = "https://openrtb-us-east-1.axonix.com/supply/prebid-server/" + axonixExt.SupplyId
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     thisURI,
		Body:    requestJSON,
		Headers: headers,
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", responseData.StatusCode),
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
		for _, bid := range seatBid.Bid {
			bid := bid
			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: getMediaType(bid.ImpID, request.Imp),
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}

	return bidResponse, nil
}

func getMediaType(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Native != nil {
				return openrtb_ext.BidTypeNative
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo
			}
			return openrtb_ext.BidTypeBanner
		}
	}
	return openrtb_ext.BidTypeBanner
}
