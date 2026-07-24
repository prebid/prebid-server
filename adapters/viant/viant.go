package viant

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// defaultCurrency is the fallback Viant bids in and the target we convert
// unsupported bid floor currencies into.
const defaultCurrency = "USD"

// supportedCurrencies are the bid floor currencies Viant can interpret directly.
// Floors already in one of these are passed through untouched so Viant applies
// its own conversion rates; anything else is converted to defaultCurrency first.
// Adding a newly supported currency here is the only change needed to stop
// converting it.
var supportedCurrencies = map[string]struct{}{
	"USD": {},
	"GBP": {},
	"CAD": {},
	"EUR": {},
	"AUD": {},
	"SAR": {},
	"AED": {},
}

type adapter struct {
	endpoint string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	reqCopy := *request
	cleanImps := make([]openrtb2.Imp, 0, len(request.Imp))

	for i := range request.Imp {
		var impExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(request.Imp[i].Ext, &impExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("invalid imp.ext for impression index %d. %s", i, err.Error()),
			})
			continue
		}

		var bidderExt openrtb_ext.ImpExtViant
		if err := jsonutil.Unmarshal(impExt.Bidder, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("invalid imp.ext.bidder for impression index %d. %s", i, err.Error()),
			})
			continue
		}

		if bidderExt.PublisherID == "" {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("imp.ext.bidder.publisherId is required for impression index %d", i),
			})
			continue
		}

		imp := request.Imp[i]
		imp.Ext = stripBidderExt(imp.Ext)

		if imp.BidFloor > 0 && imp.BidFloorCur != "" {
			if _, ok := supportedCurrencies[strings.ToUpper(imp.BidFloorCur)]; !ok {
				convertedValue, err := requestInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, defaultCurrency)
				if err != nil {
					// Viant accepts requests in currencies it doesn't support without
					// erroring and simply bids in USD. Mirror that leniency: if we can't
					// express the floor in USD, drop the floor rather than the impression
					// so Viant can still return a (USD) bid.
					errs = append(errs, &errortypes.Warning{
						Message: fmt.Sprintf("dropping unconvertible bid floor for impression index %d: %s", i, err.Error()),
					})
					imp.BidFloor = 0
					imp.BidFloorCur = ""
				} else {
					imp.BidFloorCur = defaultCurrency
					imp.BidFloor = convertedValue
				}
			}
		}

		cleanImps = append(cleanImps, imp)
	}

	if len(cleanImps) == 0 {
		return nil, append(errs, &errortypes.BadInput{
			Message: "no valid impressions in the bid request",
		})
	}

	reqCopy.Imp = cleanImps
	// Viant prices its bids in any currency it supports, so pass a supported
	// requested currency straight through and let it respond in that currency.
	// For unsupported currencies (or none) ask for USD and let Prebid core
	// convert the USD bid into the publisher's requested currency downstream.
	reqCopy.Cur = []string{resolveRequestCurrency(request.Cur)}

	requestJSON, err := json.Marshal(reqCopy)
	if err != nil {
		return nil, append(errs, err)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(reqCopy.Imp),
	}}, errs
}

// resolveRequestCurrency returns the first requested currency Viant supports so
// it can bid in it directly. If none of the requested currencies are supported
// (or none were requested), it returns defaultCurrency.
func resolveRequestCurrency(requestCurrencies []string) string {
	for _, cur := range requestCurrencies {
		if _, ok := supportedCurrencies[strings.ToUpper(cur)]; ok {
			return cur
		}
	}
	return defaultCurrency
}

// stripBidderExt removes the "bidder" and "prebid" keys from imp.ext,
// returning nil if nothing else remains.
func stripBidderExt(ext json.RawMessage) json.RawMessage {
	if ext == nil {
		return nil
	}

	var extMap map[string]json.RawMessage
	if err := jsonutil.Unmarshal(ext, &extMap); err != nil {
		return nil
	}

	delete(extMap, openrtb_ext.PrebidExtBidderKey)
	delete(extMap, openrtb_ext.PrebidExtKey)

	if len(extMap) == 0 {
		return nil
	}

	cleaned, err := json.Marshal(extMap)
	if err != nil {
		return nil
	}
	return cleaned
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info.", response.StatusCode),
		}}
	}

	if response.StatusCode == http.StatusServiceUnavailable {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("service unavailable: HTTP status %d", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info.", response.StatusCode),
		}}
	}

	if len(response.Body) == 0 {
		return nil, nil
	}

	var bidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("JSON parsing error: %s", err.Error()),
		}}
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	// Viant bids in the requested currency when it supports it, so trust the
	// currency it reports. Fall back to USD if the response omits it.
	if bidResponse.Cur != "" {
		bidderResponse.Currency = bidResponse.Cur
	} else {
		bidderResponse.Currency = defaultCurrency
	}

	var errs []error
	for _, seatBid := range bidResponse.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}

	return bidderResponse, errs
}

func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported MType %d for bid %s", bid.MType, bid.ImpID),
		}
	}
}
