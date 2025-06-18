package ogury

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: config.Endpoint}, nil
}

func (a adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	headers := buildHeaders(request)

	var errors []error
	var impsWithOguryParams []openrtb2.Imp
	for i, imp := range request.Imp {
		var impExt, impExtBidderHoist map[string]json.RawMessage
		// extract ext
		if err := jsonutil.Unmarshal(imp.Ext, &impExt); err != nil {
			return nil, append(errors, &errortypes.BadInput{
				Message: "Bidder extension not provided or can't be unmarshalled",
			})
		}
		// find Ogury bidder params
		if bidder, ok := impExt[openrtb_ext.PrebidExtBidderKey]; ok {
			if err := jsonutil.Unmarshal(bidder, &impExtBidderHoist); err != nil {
				return nil, append(errors, &errortypes.BadInput{
					Message: "Ogury bidder extension not provided or can't be unmarshalled",
				})
			}
		}

		// extract every value from imp[].ext.bidder to imp[].ext
		for key, value := range impExtBidderHoist {
			impExt[key] = value
		}
		delete(impExt, openrtb_ext.PrebidExtBidderKey)

		ext, err := jsonutil.Marshal(impExt)
		if err != nil {
			return nil, append(errors, &errortypes.BadInput{
				Message: "Error while marshaling Imp.Ext bidder extension",
			})
		}
		request.Imp[i].Ext = ext

		// save adUnitCode
		request.Imp[i].TagID = imp.ID

		// currency conversion
		// Check if imp comes with bid floor amount defined in a foreign currency
		if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != "USD" {

			// Convert to US dollars
			convertedValue, err := requestInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
			if err != nil {
				return nil, []error{err}
			}

			// Update after conversion. All imp elements inside request.Imp are shallow copies
			// therefore, their non-pointer values are not shared memory and are safe to modify.
			request.Imp[i].BidFloorCur = "USD"
			request.Imp[i].BidFloor = convertedValue
		}

		// check if imp has ogury params and filter it
		_, hasAssetKey := impExtBidderHoist["assetKey"]
		_, hasAdUnitId := impExtBidderHoist["adUnitId"]
		if hasAssetKey && hasAdUnitId {
			impsWithOguryParams = append(impsWithOguryParams, request.Imp[i])
		}
	}

	if len(impsWithOguryParams) == 0 {
		if request.Site != nil && (request.Site.Publisher == nil || request.Site.Publisher.ID == "") {
			// we can serve ads with publisherId+adunitcode combination
			return nil, []error{&errortypes.BadInput{
				Message: "Invalid request. assetKey/adUnitId or request.site.publisher.id required",
			}}
		} else if request.App != nil {
			// for app request there is no adunitcode equivalent so we can't serve ads with just the publisher id
			return nil, []error{&errortypes.BadInput{
				Message: "Invalid request. assetKey/adUnitId required",
			}}
		}
	} else if len(impsWithOguryParams) > 0 {
		request.Imp = impsWithOguryParams
	}

	requestJSON, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil

}

func buildHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	if request.Device != nil {
		headers.Add("X-Forwarded-For", request.Device.IP)
		headers.Add("X-Forwarded-For", request.Device.IPv6)
		headers.Add("User-Agent", request.Device.UA)
		headers.Add("Accept-Language", request.Device.Language)
	}
	return headers

}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported MType \"%d\", for impression \"%s\"", bid.MType, bid.ImpID),
		}
	}
}

func (a adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
	var errors []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}
	if errors != nil {
		return nil, errors
	}

	return bidResponse, nil
}
