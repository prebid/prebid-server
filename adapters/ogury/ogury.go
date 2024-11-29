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
)

type adapter struct {
	endpoint string
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: config.Endpoint,}, nil

}

func (a adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	headers := setHeaders(request)

	request.Imp = filterValidImps(request)
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "Invalid request. assetKey/adUnitId or request.site.publisher.id required",
		}}
	}

	var errors []error
	for i, imp := range request.Imp {
		var impExt, impExtBidderHoist map[string]json.RawMessage
		// extract ext
		if err := json.Unmarshal(imp.Ext, &impExt); err != nil {
			return nil, append(errors, &errortypes.BadInput{
				Message: "Bidder extension not provided or can't be unmarshalled",
			})
		}
		// find Ogury bidder params
		if bidder, ok := impExt[openrtb_ext.PrebidExtBidderKey]; ok {
			if err := json.Unmarshal(bidder, &impExtBidderHoist); err != nil {
				return nil, append(errors, &errortypes.BadInput{
					Message: "Ogury bidder extension not provided or can't be unmarshalled",
				})
			}
		}

		impExtOut := make(map[string]any, len(impExt)-1+len(impExtBidderHoist))

		// extract Ogury "bidder" params from imp.ext.bidder to imp.ext
		for key, value := range impExt {
			if key != openrtb_ext.PrebidExtBidderKey {
				impExtOut[key] = value
			}
		}
		for key, value := range impExtBidderHoist {
			impExtOut[key] = value
		}

		ext, err := json.Marshal(impExtOut)
		if err != nil {
			return nil, append(errors, &errortypes.BadInput{
				Message: "Error while marshaling Imp.Ext bidder exension",
			})
		}
		request.Imp[i].Ext = ext

		// save adUnitCode
		if adUnitCode := getAdUnitCode(impExt); adUnitCode != "" {
			request.Imp[i].TagID = adUnitCode
		} else {
			request.Imp[i].TagID = imp.ID
		}
	}

	// currency conversion
	for i, imp := range request.Imp {
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
	}

	requestJSON, err := json.Marshal(request)
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

func filterValidImps(request *openrtb2.BidRequest) (validImps []openrtb2.Imp) {
	for _, imp := range request.Imp {
		var impExt adapters.ExtImpBidder
		var impExtOgury openrtb_ext.ImpExtOgury

		if err := json.Unmarshal(imp.Ext, &impExt); err != nil {
			continue
		}
		if err := json.Unmarshal(impExt.Bidder, &impExtOgury); err != nil {
			continue
		}
		if impExtOgury.AssetKey != "" && impExtOgury.AdUnitID != "" {
			validImps = append(validImps, imp)
		}
	}

	// if we have imp with assetKey/adUnitId then we want to serve them
	if len(validImps) > 0 {
		return validImps
	}

	// no assetKey/adUnitId imps then we serve everything if publisher.ID exists
	if request.Site != nil && request.Site.Publisher.ID != "" {
		return request.Imp
	}

	// else no valid imp
	return nil
}

func getAdUnitCode(ext map[string]json.RawMessage) string {
	var prebidExt openrtb_ext.ExtImpPrebid
	v, ok := ext["prebid"]
	if !ok {
		return ""
	}

	err := json.Unmarshal(v, &prebidExt)
	if err != nil {
		return ""
	}

	return prebidExt.AdUnitCode
}

func setHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	if request.Device != nil {
		headers.Add("X-Forwarded-For", request.Device.IP)
		headers.Add("User-Agent", request.Device.UA)
		headers.Add("Accept-Language", request.Device.Language)
	}
	return headers

}

func getMediaTypeForBid(impressions []openrtb2.Imp, bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	for _, imp := range impressions {
		if imp.ID == bid.ImpID {
			switch {
			case imp.Banner != nil:
				return openrtb_ext.BidTypeBanner, nil
			case imp.Video != nil:
				return openrtb_ext.BidTypeVideo, nil
			case imp.Native != nil:
				return openrtb_ext.BidTypeNative, nil
			}
		}

	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to determine media type of impression \"%s\"", bid.ImpID),
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
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	var errors []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(request.Imp, bid)
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
