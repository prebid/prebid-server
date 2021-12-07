package valueimpression

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

type ValueImpressionAdapter struct {
	endpoint string
}

func (a *ValueImpressionAdapter) MakeRequests(request *openrtb2.BidRequest, unused *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var adapterRequests []*adapters.RequestData

	if err := preprocess(request); err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	adapterReq, err := a.makeRequest(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	adapterRequests = append(adapterRequests, adapterReq)

	return adapterRequests, errs
}

func (a *ValueImpressionAdapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, error) {
	var err error

	jsonBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    jsonBody,
		Headers: headers,
	}, nil
}

func preprocess(request *openrtb2.BidRequest) error {
	if len(request.Imp) == 0 {
		return &errortypes.BadInput{
			Message: "No Imps in Bid Request",
		}
	}
	for i := 0; i < len(request.Imp); i++ {
		var imp = &request.Imp[i]
		var bidderExt adapters.ExtImpBidder

		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		var extImp openrtb_ext.ExtImpValueImpression
		if err := json.Unmarshal(bidderExt.Bidder, &extImp); err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		imp.Ext = bidderExt.Bidder
	}

	return nil
}

// MakeBids based on valueimpression server response
func (a *ValueImpressionAdapter) MakeBids(bidRequest *openrtb2.BidRequest, unused *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Bad user input: HTTP status %d", responseData.StatusCode),
		}}
	}

	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: HTTP status %d", responseData.StatusCode),
		}}
	}

	var bidResponse openrtb2.BidResponse

	if err := json.Unmarshal(responseData.Body, &bidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	if len(bidResponse.SeatBid) == 0 {
		return nil, nil
	}

	rv := adapters.NewBidderResponseWithBidsCapacity(len(bidResponse.SeatBid[0].Bid))
	var errors []error

	for _, seatbid := range bidResponse.SeatBid {
		for _, bid := range seatbid.Bid {
			foundMatchingBid := false
			bidType := openrtb_ext.BidTypeBanner
			for _, imp := range bidRequest.Imp {
				if imp.ID == bid.ImpID {
					foundMatchingBid = true
					if imp.Banner != nil {
						bidType = openrtb_ext.BidTypeBanner
					} else if imp.Video != nil {
						bidType = openrtb_ext.BidTypeVideo
					}
					break
				}
			}

			if foundMatchingBid {
				rv.Bids = append(rv.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				})
			} else {
				errors = append(errors, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("bid id='%s' could not find valid impid='%s'", bid.ID, bid.ImpID),
				})
			}
		}
	}
	return rv, errors
}

// Builder builds a new instance of the Valueimpression adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &ValueImpressionAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
