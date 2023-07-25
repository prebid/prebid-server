package liftoff

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	Endpoint string
}

type liftoffImpressionExt struct {
	*adapters.ExtImpBidder
	// Ext represents the vungle extension.
	Ext openrtb_ext.ImpExtLiftoff `json:"vungle"`
}

type liftoffBidExt struct {
	Liftoff *bidExt `json:"liftoff,omitempty"`
}

type bidExt struct {
	AdType *int `json:"adtype,omitempty"`
}

/* Builder */

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		Endpoint: config.Endpoint,
	}

	return bidder, nil
}

/* MakeRequests */

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errs []error
	requestCopy := *request
	for _, imp := range request.Imp {
		// check if video ad
		if imp.Video == nil {
			errs = append(errs, fmt.Errorf("liftoff adapter handles video ads only"))
			continue
		}

		// Check if imp comes with bid floor amount defined in a foreign currency
		if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != "USD" {
			// Convert to US dollars
			convertedValue, err := requestInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
			if err != nil {
				return nil, []error{err}
			}

			// Update after conversion. All imp elements inside request.Imp are shallow copies
			// therefore, their non-pointer values are not shared memory and are safe to modify.
			imp.BidFloorCur = "USD"
			imp.BidFloor = convertedValue
		}

		var impExt liftoffImpressionExt
		if err := json.Unmarshal(imp.Ext, &impExt); err != nil {
			errs = append(errs, fmt.Errorf("failed unmarshalling imp ext (err)%s", err.Error()))
			continue
		}

		// get bid token & IDs
		var bidderImpExt openrtb_ext.ImpExtLiftoff
		if err := json.Unmarshal(impExt.Bidder, &bidderImpExt); err != nil {
			errs = append(errs, fmt.Errorf("failed unmarshalling bidder imp ext (err)%s", err.Error()))
			continue
		}

		if len(bidderImpExt.PubAppStoreID) > 0 && len(bidderImpExt.PlacementRefID) > 0 && len(bidderImpExt.BidToken) > 0 {
			impExt.Ext = bidderImpExt
			if newImpExt, err := json.Marshal(impExt); err == nil {
				imp.Ext = newImpExt
			} else {
				errs = append(errs, fmt.Errorf("failed re-marshalling imp ext"))
				continue
			}

			imp.TagID = bidderImpExt.PlacementRefID
			requestCopy.Imp = []openrtb2.Imp{imp}
			// must make a shallow copy for pointers.
			requestAppCopy := *request.App
			requestAppCopy.ID = bidderImpExt.PubAppStoreID
			requestCopy.App = &requestAppCopy
		} else {
			errs = append(errs, &errortypes.BadInput{Message: "app_store_id and placement_reference_ID and bid token should all be provided"})
			continue
		}

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
				"Content-Type": []string{"application/json"},
			},
		}

		requests = append(requests, requestData)
	}

	return requests, errs
}

/* MakeBids */

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
