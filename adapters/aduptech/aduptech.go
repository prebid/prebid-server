package aduptech

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint        string
	target_currency string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint:        config.Endpoint,
		target_currency: "EUR",
	}

	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	for i := range request.Imp {
		imp := &request.Imp[i]
		// Check if imp comes with bid floor amount defined in a foreign currency
		if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != a.target_currency {

			convertedValue, err := a.convertCurrency(imp.BidFloor, imp.BidFloorCur, reqInfo)
			if err != nil {
				return nil, err
			}

			imp.BidFloorCur = a.target_currency
			imp.BidFloor = convertedValue
		}
	}

	requestJSON, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint,
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) convertCurrency(value float64, cur string, reqInfo *adapters.ExtraRequestInfo) (float64, []error) {
	convertedValue, err := reqInfo.ConvertCurrency(value, cur, a.target_currency)

	if err != nil {
		var convErr currency.ConversionNotFoundError
		if errors.As(err, &convErr) {

			// try again by first converting to USD
			// then convert to target_currency
			convertedValue, err = reqInfo.ConvertCurrency(value, cur, "USD")

			if err != nil {
				return 0, []error{err}
			}

			convertedValue, err = reqInfo.ConvertCurrency(convertedValue, "USD", a.target_currency)

			if err != nil {
				return 0, []error{err}
			}
		} else {
			return 0, []error{err}
		}
	}
	return convertedValue, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Run with request.debug = 1 for more info.",
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
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getBidType(bid.MType)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

func getBidType(markupType openrtb2.MarkupType) (openrtb_ext.BidType, error) {
	switch markupType {
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unknown markup type: %d", markupType),
		}
	}
}
