package apacdex

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

type apacdexAdapter struct {
	endpoint string
}

// Builder builds a new instance of the Apacdex adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &apacdexAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *apacdexAdapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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

func (a *apacdexAdapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, error) {
	var err error

	jsonBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

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

		var extImp openrtb_ext.ExtImpApacdex
		if err := json.Unmarshal(bidderExt.Bidder, &extImp); err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		imp.Ext = bidderExt.Bidder
	}

	return nil
}

func (a *apacdexAdapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	if len(response.SeatBid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	var errors []error

	for _, seatbid := range response.SeatBid {
		for _, bid := range seatbid.Bid {
			imp, err := getImpressionForBid(request.Imp, bid.ImpID)
			if err != nil {
				errors = append(errors, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("bid id='%s' could not find valid impid='%s'", bid.ID, bid.ImpID),
				})
				continue
			}

			bidType := openrtb_ext.BidTypeBanner
			if imp.Banner != nil {
				bidType = openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				bidType = openrtb_ext.BidTypeVideo
			} else if imp.Audio != nil {
				bidType = openrtb_ext.BidTypeAudio
			} else if imp.Native != nil {
				bidType = openrtb_ext.BidTypeNative
			} else {
				errors = append(errors, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unknown bidType for bid id='%s'", bid.ID),
				})
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}
	return bidResponse, errors
}

func getImpressionForBid(imps []openrtb2.Imp, impID string) (openrtb2.Imp, error) {
	result := openrtb2.Imp{}
	found := false
	for _, imp := range imps {
		if imp.ID == impID {
			result = imp
			found = true
			break
		}
	}
	if found {
		return result, nil
	}
	return result, fmt.Errorf("not found impression matched with ImpID=%s", impID)
}
