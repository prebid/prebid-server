package frvradn

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"strings"
)

type adapter struct {
	uri string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	if config.Endpoint == "" {
		return nil, errors.New("missing endpoint adapter parameter")
	}

	bidder := &adapter{
		uri: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errs []error

	requestCopy := *request
	for _, imp := range request.Imp {
		frvrAdnExt, err := getImpressionExt(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != "USD" {
			convertedValue, err := requestInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
			if err != nil {
				errs = append(errs, err)
				continue
			}
			imp.BidFloorCur = "USD"
			imp.BidFloor = convertedValue
		}

		ext, err := json.Marshal(frvrAdnExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		imp.Ext = ext

		requestCopy.Imp = []openrtb2.Imp{imp}

		requestJSON, err := json.Marshal(requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requestData := &adapters.RequestData{
			Method: "POST",
			Uri:    a.uri,
			Body:   requestJSON,
		}
		requests = append(requests, requestData)
	}
	return requests, errs
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

	if len(responseData.Body) == 0 {
		return nil, nil
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	var errs []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getBidMediaType(bid.ImpID, request.Imp)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, errs
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ImpExtFRVRAdn, error) {
	var extImpBidder adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &extImpBidder); err != nil {
		return nil, &errortypes.BadInput{
			Message: "missing ext",
		}
	}
	var frvrAdnExt openrtb_ext.ImpExtFRVRAdn
	if err := json.Unmarshal(extImpBidder.Bidder, &frvrAdnExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "missing ext.bidder",
		}
	}

	if len(frvrAdnExt.PublisherID) == 0 || len(frvrAdnExt.AdUnitID) == 0 {
		return nil, &errortypes.BadInput{
			Message: "publisher_id and ad_unit_id are required",
		}
	}
	return &frvrAdnExt, nil
}

func getBidMediaType(impId string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}
			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			}
			if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			}
			return "", fmt.Errorf("imp %v with unknown media type", impId)
		}
	}
	return "", fmt.Errorf("unknown imp id: %s", impId)
}
