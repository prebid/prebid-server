package liftoff

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const SupportedCurrency = "USD"

type adapter struct {
	Endpoint string
}

type liftoffImpressionExt struct {
	*adapters.ExtImpBidder
	// Ext represents the vungle extension.
	Ext openrtb_ext.ImpExtLiftoff `json:"vungle"`
}

// Builder builds a new instance of the Liftoff adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{Endpoint: config.Endpoint}, nil
}

// MakeRequests split impressions into bid requests and change them into the form that liftoff can handle.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errs []error
	requestCopy := *request
	for _, imp := range request.Imp {
		// Check if imp comes with bid floor amount defined in a foreign currency
		if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != SupportedCurrency {
			// Convert to US dollars
			convertedValue, err := requestInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, SupportedCurrency)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to convert currency (err)%s", err.Error()))
				continue
			}

			// Update after conversion. All imp elements inside request.Imp are shallow copies
			// therefore, their non-pointer values are not shared memory and are safe to modify.
			imp.BidFloorCur = SupportedCurrency
			imp.BidFloor = convertedValue
		}

		var impExt liftoffImpressionExt
		if err := json.Unmarshal(imp.Ext, &impExt); err != nil {
			errs = append(errs, fmt.Errorf("failed unmarshalling imp ext (err)%s", err.Error()))
			continue
		}

		// get placement_reference_id & pub_app_store_id
		var bidderImpExt openrtb_ext.ImpExtLiftoff
		if err := json.Unmarshal(impExt.Bidder, &bidderImpExt); err != nil {
			errs = append(errs, fmt.Errorf("failed unmarshalling bidder imp ext (err)%s", err.Error()))
			continue
		}

		bidderImpExt.BidToken = requestCopy.User.BuyerUID
		impExt.Ext = bidderImpExt
		if newImpExt, err := json.Marshal(impExt); err == nil {
			imp.Ext = newImpExt
		} else {
			errs = append(errs, errors.New("failed re-marshalling imp ext"))
			continue
		}

		imp.TagID = bidderImpExt.PlacementRefID
		requestCopy.Imp = []openrtb2.Imp{imp}
		// must make a shallow copy for pointers.
		requestAppCopy := *request.App
		requestAppCopy.ID = bidderImpExt.PubAppStoreID
		requestCopy.App = &requestAppCopy

		requestJSON, err := json.Marshal(requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requestData := &adapters.RequestData{
			Method: "POST",
			Uri:    a.Endpoint,
			Body:   requestJSON,
			Headers: http.Header{
				"Content-Type":      []string{"application/json"},
				"Accept":            []string{"application/json"},
				"X-OpenRTB-Version": []string{"2.5"},
			},
		}

		requests = append(requests, requestData)
	}

	return requests, errs
}

// MakeBids collect bid response from liftoff and change them into the form that Prebid Server can handle.
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

	var errs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for _, temp := range seatBid.Bid {
			bid := temp // avoid taking address of a for loop variable
			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: openrtb_ext.BidTypeVideo,
			}

			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}

	return bidResponse, errs
}