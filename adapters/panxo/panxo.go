package panxo

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint string
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "no impressions in request",
		}}
	}

	// Group by property without changing the original request. This mirrors the
	// Client adapter and keeps a mixed-property auction usable instead of
	// rejecting every impression because one placement has a different key.
	type requestGroup struct {
		propertyKey string
		imps        []openrtb2.Imp
	}
	groups := make(map[string]*requestGroup)
	groupOrder := make([]string, 0)
	var conversionErrors []error

	for i := range request.Imp {
		impPropertyKey, err := propertyKeyFromImpExt(request.Imp[i].Ext, i)
		if err != nil {
			return nil, []error{err}
		}

		// Panxo's auction and billing amounts are USD. Convert copies of both
		// open and deal floors before comparing them to those amounts.
		impCopy, err := impressionWithUSDFloors(request.Imp[i], reqInfo)
		if err != nil {
			conversionErrors = append(conversionErrors, &errortypes.BadInput{
				Message: fmt.Sprintf("cannot normalize floors for impression index %d: %s", i, err),
			})
			continue
		}

		group, exists := groups[impPropertyKey]
		if !exists {
			group = &requestGroup{propertyKey: impPropertyKey}
			groups[impPropertyKey] = group
			groupOrder = append(groupOrder, impPropertyKey)
		}

		// Extract placement routing before omitting all impression extensions.
		// Extensions can contain arbitrary publisher-owned identity.
		impCopy.Ext = nil
		group.imps = append(group.imps, impCopy)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	requests := make([]*adapters.RequestData, 0, len(groupOrder))
	for _, propertyKey := range groupOrder {
		group := groups[propertyKey]
		requestCopy := *request
		requestCopy.Imp = group.imps
		requestCopy.Ext = nil
		// An omitted cur already means USD. Preserve the original auction
		// currency so PBS core can convert the returned USD CPM if needed.
		if len(request.Cur) != 0 {
			requestCopy.Cur = []string{"USD"}
		}
		var err error
		if request.Site != nil {
			siteCopy := *request.Site
			siteCopy.Page = ""
			siteCopy.Ref = ""
			siteCopy.Ext, err = inventoryExtWithPanxoSignal(request.Site.Ext)
			if err != nil {
				return nil, []error{err}
			}
			requestCopy.Site = &siteCopy
		}
		if request.App != nil {
			appCopy := *request.App
			appCopy.Ext, err = inventoryExtWithPanxoSignal(request.App.Ext)
			if err != nil {
				return nil, []error{err}
			}
			requestCopy.App = &appCopy
		}
		requestCopy.User, err = consentOnlyUser(request.User)
		if err != nil {
			return nil, []error{err}
		}
		requestCopy.Device, err = withoutDeviceIdentity(request.Device)
		if err != nil {
			return nil, []error{err}
		}

		reqJSON, err := jsonutil.Marshal(&requestCopy)
		if err != nil {
			return nil, []error{err}
		}

		requests = append(requests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     fmt.Sprintf("%s?key=%s&source=prebid-server", a.endpoint, url.QueryEscape(group.propertyKey)),
			Body:    reqJSON,
			Headers: headers.Clone(),
			ImpIDs:  openrtb_ext.GetImpIDs(group.imps),
		})
	}

	return requests, conversionErrors
}

func impressionWithUSDFloors(imp openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) (openrtb2.Imp, error) {
	var err error
	imp.BidFloor, err = floorInUSD(imp.BidFloor, imp.BidFloorCur, reqInfo)
	if err != nil {
		return imp, err
	}
	if imp.BidFloorCur != "" {
		imp.BidFloorCur = "USD"
	}
	if imp.PMP != nil {
		pmp := *imp.PMP
		pmp.Deals = append([]openrtb2.Deal(nil), imp.PMP.Deals...)
		for i := range pmp.Deals {
			pmp.Deals[i].BidFloor, err = floorInUSD(pmp.Deals[i].BidFloor, pmp.Deals[i].BidFloorCur, reqInfo)
			if err != nil {
				return imp, err
			}
			if pmp.Deals[i].BidFloorCur != "" {
				pmp.Deals[i].BidFloorCur = "USD"
			}
		}
		imp.PMP = &pmp
	}
	return imp, nil
}

func floorInUSD(amount float64, from string, reqInfo *adapters.ExtraRequestInfo) (float64, error) {
	if amount < 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0, fmt.Errorf("invalid bid floor")
	}
	from = strings.ToUpper(strings.TrimSpace(from))
	if amount == 0 || from == "" || from == "USD" {
		return amount, nil
	}
	if reqInfo == nil || reqInfo.CurrencyConversions == nil {
		return 0, fmt.Errorf("currency conversion unavailable")
	}
	converted, err := reqInfo.ConvertCurrency(amount, from, "USD")
	if err != nil || converted <= 0 || math.IsNaN(converted) || math.IsInf(converted, 0) {
		return 0, fmt.Errorf("cannot convert positive bid floor to USD")
	}
	return converted, nil
}

// Copy inventory extensions while limiting the Panxo namespace to the
// ephemeral AI marker. Other extension namespaces remain forward-compatible.
func inventoryExtWithPanxoSignal(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var ext map[string]json.RawMessage
	if err := jsonutil.Unmarshal(raw, &ext); err != nil {
		return nil, &errortypes.BadInput{Message: "invalid inventory.ext"}
	}
	if ext == nil {
		return nil, nil
	}
	if rawData, exists := ext["data"]; exists {
		var data map[string]json.RawMessage
		if err := jsonutil.Unmarshal(rawData, &data); err != nil || data == nil {
			delete(ext, "data")
		} else {
			if rawPanxo, exists := data["panxo"]; exists {
				delete(data, "panxo")
				var panxo map[string]json.RawMessage
				if err := jsonutil.Unmarshal(rawPanxo, &panxo); err == nil {
					var ai *bool
					if rawAI, exists := panxo["ai"]; exists {
						if err := jsonutil.Unmarshal(rawAI, &ai); err == nil && ai != nil {
							data["panxo"], _ = jsonutil.Marshal(map[string]bool{"ai": *ai})
						}
					}
				}
			}
			ext["data"], _ = jsonutil.Marshal(data)
		}
	}
	return jsonutil.Marshal(ext)
}

// consentOnlyUser retains existing OpenRTB privacy fields, not user identity,
// memberships or arbitrary user extensions. This is not a new decision DTO.
func consentOnlyUser(user *openrtb2.User) (*openrtb2.User, error) {
	if user == nil {
		return nil, nil
	}
	result := &openrtb2.User{Consent: user.Consent}
	if len(user.Ext) != 0 {
		var ext map[string]json.RawMessage
		if err := jsonutil.Unmarshal(user.Ext, &ext); err != nil {
			return nil, &errortypes.BadInput{Message: "invalid user.ext"}
		}
		if raw, exists := ext["consent"]; exists {
			var consent string
			if err := jsonutil.Unmarshal(raw, &consent); err == nil && consent != "" {
				result.Ext, _ = jsonutil.Marshal(map[string]string{"consent": consent})
			}
		}
	}
	return result, nil
}

// Remove named device identifiers and all device extensions.
// Validate discarded extension shapes without including their contents in errors.
func withoutDeviceIdentity(device *openrtb2.Device) (*openrtb2.Device, error) {
	if device == nil {
		return nil, nil
	}
	result := *device
	result.IFA = ""
	result.DIDSHA1 = ""
	result.DIDMD5 = ""
	result.DPIDSHA1 = ""
	result.DPIDMD5 = ""
	result.MACSHA1 = ""
	result.MACMD5 = ""
	result.IP = ""
	result.IPv6 = ""
	if len(device.Ext) != 0 {
		var ext map[string]json.RawMessage
		if err := jsonutil.Unmarshal(device.Ext, &ext); err != nil {
			return nil, &errortypes.BadInput{Message: "invalid device.ext"}
		}
	}
	result.Ext = nil
	return &result, nil
}

func propertyKeyFromImpExt(impExt json.RawMessage, index int) (string, error) {
	var extBidder adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(impExt, &extBidder); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("invalid imp.ext for impression index %d: malformed object", index),
		}
	}
	var bidderExt openrtb_ext.ExtImpPanxo
	if err := jsonutil.Unmarshal(extBidder.Bidder, &bidderExt); err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("invalid imp.ext.bidder for impression index %d: malformed object", index),
		}
	}
	propertyKey := strings.TrimSpace(bidderExt.PropertyKey)
	if propertyKey == "" {
		return "", &errortypes.BadInput{Message: "propertyKey is required"}
	}
	return propertyKey, nil
}

func (*adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidCount := 0
	for _, seatBid := range bidResp.SeatBid {
		bidCount += len(seatBid.Bid)
	}
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(bidCount)
	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	}

	// prepareStoredResponse in exchange/bidder.go creates RequestData with an
	// empty URI and no ImpIDs. Its impression was removed from the request;
	// core handles optional impid replacement after MakeBids. Require explicit
	// mtype here and retain impression membership checks for HTTP responses.
	storedResponse := requestData != nil && requestData.Uri == "" && len(requestData.ImpIDs) == 0

	var errs []error
	requestedImpIDs := make(map[string]struct{})
	if requestData != nil {
		for _, impID := range requestData.ImpIDs {
			requestedImpIDs[impID] = struct{}{}
		}
	}
	requestedImps := make(map[string]*openrtb2.Imp)
	if request != nil && !storedResponse {
		for i := range request.Imp {
			imp := &request.Imp[i]
			if len(requestedImpIDs) == 0 {
				requestedImps[imp.ID] = imp
				continue
			}
			if _, included := requestedImpIDs[imp.ID]; included {
				requestedImps[imp.ID] = imp
			}
		}
	}

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bid := &sb.Bid[i]
			bidType, err := getBidType(bid, requestedImps[bid.ImpID], storedResponse)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				// Keep the complete OpenRTB bid intact. Prebid Server promotes
				// bid.ext.prebid.meta into its canonical response metadata and
				// preserves other extensions such as DSA and future DSP fields.
				Bid:     bid,
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

func getBidType(bid *openrtb2.Bid, imp *openrtb2.Imp, storedResponse bool) (openrtb_ext.BidType, error) {
	if imp == nil && !storedResponse {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("bid %s references an impression not sent in this request: %s", bid.ID, bid.ImpID),
		}
	}

	var bidType openrtb_ext.BidType
	switch bid.MType {
	case openrtb2.MarkupBanner:
		bidType = openrtb_ext.BidTypeBanner
	case openrtb2.MarkupVideo:
		bidType = openrtb_ext.BidTypeVideo
	case openrtb2.MarkupNative:
		bidType = openrtb_ext.BidTypeNative
	case 0:
		if imp == nil {
			return "", &errortypes.BadServerResponse{
				Message: fmt.Sprintf("bid %s omitted mtype for a stored response without impression context", bid.ID),
			}
		}
		requestedTypes := make([]openrtb_ext.BidType, 0, 3)
		if imp.Banner != nil {
			requestedTypes = append(requestedTypes, openrtb_ext.BidTypeBanner)
		}
		if imp.Video != nil {
			requestedTypes = append(requestedTypes, openrtb_ext.BidTypeVideo)
		}
		if imp.Native != nil {
			requestedTypes = append(requestedTypes, openrtb_ext.BidTypeNative)
		}
		if len(requestedTypes) != 1 {
			return "", &errortypes.BadServerResponse{
				Message: fmt.Sprintf("bid %s omitted mtype for an impression with %d supported media types", bid.ID, len(requestedTypes)),
			}
		}
		bidType = requestedTypes[0]
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported mtype %d for bid %s", bid.MType, bid.ID),
		}
	}

	if imp != nil && ((bidType == openrtb_ext.BidTypeBanner && imp.Banner == nil) ||
		(bidType == openrtb_ext.BidTypeVideo && imp.Video == nil) ||
		(bidType == openrtb_ext.BidTypeNative && imp.Native == nil)) {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("bid %s returned %s for an impression that did not request it", bid.ID, bidType),
		}
	}

	return bidType, nil
}
