package aduptech

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	prebidcurrency "github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"golang.org/x/text/currency"
)

type adapter struct {
	endpoint  string
	extraInfo ExtraInfo
}

type ExtraInfo struct {
	TargetCurrency string `json:"target_currency,omitempty"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	var extraInfo ExtraInfo
	if err := jsonutil.Unmarshal([]byte(config.ExtraAdapterInfo), &extraInfo); err != nil {
		return nil, fmt.Errorf("invalid extra info: %w", err)
	}

	if extraInfo.TargetCurrency == "" {
		return nil, errors.New("invalid extra info: TargetCurrency is empty, pls check")
	}

	parsedCurrency, err := currency.ParseISO(extraInfo.TargetCurrency)
	if err != nil {
		return nil, fmt.Errorf("invalid extra info: invalid TargetCurrency %s, pls check", extraInfo.TargetCurrency)
	}
	extraInfo.TargetCurrency = parsedCurrency.String()

	bidder := &adapter{
		endpoint:  config.Endpoint,
		extraInfo: extraInfo,
	}

	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	for i := range request.Imp {
		imp := &request.Imp[i]
		// Check if imp comes with bid floor amount defined in a foreign currency
		if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != a.extraInfo.TargetCurrency {

			convertedValue, err := a.convertCurrency(imp.BidFloor, imp.BidFloorCur, reqInfo)
			if err != nil {
				return nil, []error{err}
			}

			imp.BidFloorCur = a.extraInfo.TargetCurrency
			imp.BidFloor = convertedValue
		}
	}

	requestJSON, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: http.MethodPost,
		Uri:    a.endpoint,
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

// convertCurrency attempts to convert a given value from the specified currency (cur) to the
// target currency specified in the adapter's extraInfo. If the conversion directly to the
// target currency fails due to a ConversionNotFoundError, it attempts an intermediate conversion
// through USD. Returns the converted value or an error if conversion fails.
func (a *adapter) convertCurrency(value float64, cur string, reqInfo *adapters.ExtraRequestInfo) (float64, error) {
	convertedValue, err := reqInfo.ConvertCurrency(value, cur, a.extraInfo.TargetCurrency)

	if err != nil {
		var convErr prebidcurrency.ConversionNotFoundError
		if !errors.As(err, &convErr) {
			return 0, err
		}

		// try again by first converting to USD
		// then convert to target_currency
		convertedValue, err = reqInfo.ConvertCurrency(value, cur, "USD")

		if err != nil {
			return 0, fmt.Errorf("Currency conversion rate not found from '%s' to '%s'. Error converting from '%s' to 'USD': %w", cur, a.extraInfo.TargetCurrency, cur, err)
		}

		convertedValue, err = reqInfo.ConvertCurrency(convertedValue, "USD", a.extraInfo.TargetCurrency)

		if err != nil {
			return 0, fmt.Errorf("Currency conversion rate not found from '%s' to '%s'. Error converting from 'USD' to '%s': %w", cur, a.extraInfo.TargetCurrency, a.extraInfo.TargetCurrency, err)
		}
	}
	return convertedValue, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur

	var errs []error
	for i := range response.SeatBid {
		seatBid := &response.SeatBid[i]
		for j := range seatBid.Bid {
			bid := &seatBid.Bid[j]
			bidType, err := getBidType(bid.MType)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
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
