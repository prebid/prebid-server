package apacdex

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Apacdex adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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

func (a *adapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, error) {
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
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
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

		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		var extImp openrtb_ext.ExtImpApacdex
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &extImp); err != nil {
			return &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		imp.Ext = bidderExt.Bidder
	}

	return nil
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

	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
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
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errors = append(errors, err)
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

// getMediaTypeForBid determines which type of bid.
func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err := jsonutil.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			return openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to parse bid mediatype for impression \"%s\"", bid.ImpID),
	}
}
