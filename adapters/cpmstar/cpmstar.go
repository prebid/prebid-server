package cpmstar

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type Adapter struct {
	endpoint string
}

func (a *Adapter) MakeRequests(request *openrtb.BidRequest, unused *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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

func (a *Adapter) makeRequest(request *openrtb.BidRequest) (*adapters.RequestData, error) {
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

func preprocess(request *openrtb.BidRequest) error {
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

		if err := validateImp(imp); err != nil {
			return err
		}

		var extImp openrtb_ext.ExtImpCpmstar
		if err := json.Unmarshal(bidderExt.Bidder, &extImp); err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		imp.Ext = bidderExt.Bidder
	}

	return nil
}

func validateImp(imp *openrtb.Imp) error {
	if imp.Banner == nil && imp.Video == nil {
		return &errortypes.BadInput{
			Message: "Only Banner and Video bid-types are supported at this time",
		}
	}
	return nil
}

// MakeBids based on cpmstar server response
func (a *Adapter) MakeBids(bidRequest *openrtb.BidRequest, unused *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected HTTP status code: %d. Run with request.debug = 1 for more info", responseData.StatusCode),
		}}
	}

	var bidResponse openrtb.BidResponse

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

func NewCpmstarBidder(endpoint string) *Adapter {
	return &Adapter{
		endpoint: endpoint,
	}
}
