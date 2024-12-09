package frvradn

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
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
			ImpIDs: openrtb_ext.GetImpIDs(requestCopy.Imp),
		}
		requests = append(requests, requestData)
	}
	return requests, errs
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	if len(responseData.Body) == 0 {
		return nil, nil
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getBidMediaType(&seatBid.Bid[i])
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
	if err := jsonutil.Unmarshal(imp.Ext, &extImpBidder); err != nil {
		return nil, &errortypes.BadInput{
			Message: "missing ext",
		}
	}
	var frvrAdnExt openrtb_ext.ImpExtFRVRAdn
	if err := jsonutil.Unmarshal(extImpBidder.Bidder, &frvrAdnExt); err != nil {
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

func getBidMediaType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	var extBid openrtb_ext.ExtBid
	err := jsonutil.Unmarshal(bid.Ext, &extBid)
	if err != nil {
		return "", fmt.Errorf("unable to deserialize imp %v bid.ext", bid.ImpID)
	}

	if extBid.Prebid == nil {
		return "", fmt.Errorf("imp %v with unknown media type", bid.ImpID)
	}

	return extBid.Prebid.Type, nil
}
