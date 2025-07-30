package pubmatic

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
)

const MAX_IMPRESSIONS_PUBMATIC = 30

const ae = "ae"

type PubmaticAdapter struct {
	URI        string
	bidderName string
}

type pubmaticBidExt struct {
	VideoCreativeInfo  *pubmaticBidExtVideo `json:"video,omitempty"`
	Marketplace        string               `json:"marketplace,omitempty"`
	PrebidDealPriority int                  `json:"prebiddealpriority,omitempty"`
	InBannerVideo      bool                 `json:"ibv,omitempty"`
}

type pubmaticWrapperExt struct {
	ProfileID int `json:"profile,omitempty"`
	VersionID int `json:"version,omitempty"`
}

type pubmaticBidExtVideo struct {
	Duration *int `json:"duration,omitempty"`
}

type ExtImpBidderPubmatic struct {
	adapters.ExtImpBidder
	Data json.RawMessage `json:"data,omitempty"`
	AE   int             `json:"ae,omitempty"`
	GpId string          `json:"gpid,omitempty"`
}

type ExtAdServer struct {
	Name   string `json:"name"`
	AdSlot string `json:"adslot"`
}

type marketplaceReqExt struct {
	AllowedBidders []string `json:"allowedbidders,omitempty"`
}

type extRequestAdServer struct {
	Wrapper     *pubmaticWrapperExt `json:"wrapper,omitempty"`
	Acat        []string            `json:"acat,omitempty"`
	Marketplace *marketplaceReqExt  `json:"marketplace,omitempty"`
}

type respExt struct {
	FledgeAuctionConfigs map[string]json.RawMessage `json:"fledge_auction_configs,omitempty"`
}

const (
	dctrKeyName        = "key_val"
	pmZoneIDKeyName    = "pmZoneId"
	pmZoneIDKeyNameOld = "pmZoneID"
	ImpExtAdUnitKey    = "dfp_ad_unit_code"
	AdServerGAM        = "gam"
	AdServerKey        = "adserver"
	PBAdslotKey        = "pbadslot"
	gpIdKey            = "gpid"
)

func (a *PubmaticAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	pubID := ""
	extractWrapperExtFromImp := true
	extractPubIDFromImp := true

	displayManager, displayManagerVer := "", ""
	if request.App != nil && request.App.Ext != nil {
		displayManager, displayManagerVer = getDisplayManagerAndVer(request.App)
	}

	newReqExt, err := extractPubmaticExtFromRequest(request)
	if err != nil {
		return nil, []error{err}
	}
	wrapperExt := newReqExt.Wrapper
	if wrapperExt != nil && wrapperExt.ProfileID != 0 && wrapperExt.VersionID != 0 {
		extractWrapperExtFromImp = false
	}

	for i := 0; i < len(request.Imp); i++ {
		wrapperExtFromImp, pubIDFromImp, err := parseImpressionObject(&request.Imp[i], extractWrapperExtFromImp, extractPubIDFromImp, displayManager, displayManagerVer)

		// If the parsing is failed, remove imp and add the error.
		if err != nil {
			errs = append(errs, err)
			request.Imp = append(request.Imp[:i], request.Imp[i+1:]...)
			i--
			continue
		}

		if extractWrapperExtFromImp {
			if wrapperExtFromImp != nil {
				if wrapperExt == nil {
					wrapperExt = &pubmaticWrapperExt{}
				}
				if wrapperExt.ProfileID == 0 {
					wrapperExt.ProfileID = wrapperExtFromImp.ProfileID
				}
				if wrapperExt.VersionID == 0 {
					wrapperExt.VersionID = wrapperExtFromImp.VersionID
				}

				if wrapperExt != nil && wrapperExt.ProfileID != 0 && wrapperExt.VersionID != 0 {
					extractWrapperExtFromImp = false
				}
			}
		}

		if extractPubIDFromImp && pubIDFromImp != "" {
			pubID = pubIDFromImp
			extractPubIDFromImp = false
		}
	}

	// If all the requests are invalid, Call to adaptor is skipped
	if len(request.Imp) == 0 {
		return nil, errs
	}

	newReqExt.Wrapper = wrapperExt
	rawExt, err := json.Marshal(newReqExt)
	if err != nil {
		return nil, []error{err}
	}
	request.Ext = rawExt

	if request.Site != nil {
		siteCopy := *request.Site
		if siteCopy.Publisher != nil {
			publisherCopy := *siteCopy.Publisher
			publisherCopy.ID = pubID
			siteCopy.Publisher = &publisherCopy
		} else {
			siteCopy.Publisher = &openrtb2.Publisher{ID: pubID}
		}
		request.Site = &siteCopy
	} else if request.App != nil {
		appCopy := *request.App
		if appCopy.Publisher != nil {
			publisherCopy := *appCopy.Publisher
			publisherCopy.ID = pubID
			appCopy.Publisher = &publisherCopy
		} else {
			appCopy.Publisher = &openrtb2.Publisher{ID: pubID}
		}
		request.App = &appCopy
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.URI,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, errs
}

// validateAdslot validate the optional adslot string
// valid formats are 'adslot@WxH', 'adslot' and no adslot
func validateAdSlot(adslot string, imp *openrtb2.Imp) error {
	adSlotStr := strings.TrimSpace(adslot)

	if len(adSlotStr) == 0 {
		return nil
	}

	if !strings.Contains(adSlotStr, "@") {
		imp.TagID = adSlotStr
		return nil
	}

	adSlot := strings.Split(adSlotStr, "@")
	if len(adSlot) == 2 && adSlot[0] != "" && adSlot[1] != "" {
		imp.TagID = strings.TrimSpace(adSlot[0])

		adSize := strings.Split(strings.ToLower(adSlot[1]), "x")
		if len(adSize) != 2 {
			return fmt.Errorf("Invalid size provided in adSlot %v", adSlotStr)
		}

		width, err := strconv.Atoi(strings.TrimSpace(adSize[0]))
		if err != nil {
			return fmt.Errorf("Invalid width provided in adSlot %v", adSlotStr)
		}

		heightStr := strings.Split(adSize[1], ":")
		height, err := strconv.Atoi(strings.TrimSpace(heightStr[0]))
		if err != nil {
			return fmt.Errorf("Invalid height provided in adSlot %v", adSlotStr)
		}

		//In case of video, size could be derived from the player size
		if imp.Banner != nil {
			imp.Banner = assignBannerWidthAndHeight(imp.Banner, int64(width), int64(height))
		}
	} else {
		return fmt.Errorf("Invalid adSlot %v", adSlotStr)
	}

	return nil
}

func assignBannerSize(banner *openrtb2.Banner) (*openrtb2.Banner, error) {
	if banner.W != nil && banner.H != nil {
		return banner, nil
	}

	return assignBannerWidthAndHeight(banner, banner.Format[0].W, banner.Format[0].H), nil
}

func assignBannerWidthAndHeight(banner *openrtb2.Banner, w, h int64) *openrtb2.Banner {
	bannerCopy := *banner
	bannerCopy.W = ptrutil.ToPtr(w)
	bannerCopy.H = ptrutil.ToPtr(h)
	return &bannerCopy
}

// parseImpressionObject parse the imp to get it ready to send to pubmatic
func parseImpressionObject(imp *openrtb2.Imp, extractWrapperExtFromImp, extractPubIDFromImp bool, displayManager, displayManagerVer string) (*pubmaticWrapperExt, string, error) {
	var wrapExt *pubmaticWrapperExt
	var pubID string

	// PubMatic supports banner and video impressions.
	if imp.Banner == nil && imp.Video == nil && imp.Native == nil {
		return wrapExt, pubID, fmt.Errorf("invalid MediaType. PubMatic only supports Banner, Video and Native. Ignoring ImpID=%s", imp.ID)
	}

	if imp.Audio != nil {
		imp.Audio = nil
	}

	// Populate imp.displaymanager and imp.displaymanagerver if the SDK failed to do it.
	if imp.DisplayManager == "" && imp.DisplayManagerVer == "" && displayManager != "" && displayManagerVer != "" {
		imp.DisplayManager = displayManager
		imp.DisplayManagerVer = displayManagerVer
	}

	var bidderExt ExtImpBidderPubmatic
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return wrapExt, pubID, err
	}

	var pubmaticExt openrtb_ext.ExtImpPubmatic
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &pubmaticExt); err != nil {
		return wrapExt, pubID, err
	}

	if extractPubIDFromImp {
		pubID = strings.TrimSpace(pubmaticExt.PublisherId)
	}

	// Parse Wrapper Extension only once per request
	if extractWrapperExtFromImp && len(pubmaticExt.WrapExt) != 0 {
		err := jsonutil.Unmarshal([]byte(pubmaticExt.WrapExt), &wrapExt)
		if err != nil {
			return wrapExt, pubID, fmt.Errorf("Error in Wrapper Parameters = %v  for ImpID = %v WrapperExt = %v", err.Error(), imp.ID, string(pubmaticExt.WrapExt))
		}
	}

	if err := validateAdSlot(strings.TrimSpace(pubmaticExt.AdSlot), imp); err != nil {
		return wrapExt, pubID, err
	}

	if imp.Banner != nil {
		bannerCopy, err := assignBannerSize(imp.Banner)
		if err != nil {
			return wrapExt, pubID, err
		}
		imp.Banner = bannerCopy
	}

	if pubmaticExt.Kadfloor != "" {
		bidfloor, err := strconv.ParseFloat(strings.TrimSpace(pubmaticExt.Kadfloor), 64)
		if err == nil {
			// In case of valid kadfloor, select maximum of original imp.bidfloor and kadfloor
			imp.BidFloor = math.Max(bidfloor, imp.BidFloor)
		}
	}

	extMap := make(map[string]interface{}, 0)
	if pubmaticExt.Keywords != nil && len(pubmaticExt.Keywords) != 0 {
		addKeywordsToExt(pubmaticExt.Keywords, extMap)
	}
	//Give preference to direct values of 'dctr' & 'pmZoneId' params in extension
	if pubmaticExt.Dctr != "" {
		extMap[dctrKeyName] = pubmaticExt.Dctr
	}
	if pubmaticExt.PmZoneID != "" {
		extMap[pmZoneIDKeyName] = pubmaticExt.PmZoneID
	}

	if len(bidderExt.Data) > 0 {
		populateFirstPartyDataImpAttributes(bidderExt.Data, extMap)
	}

	if bidderExt.AE != 0 {
		extMap[ae] = bidderExt.AE
	}

	if bidderExt.GpId != "" {
		extMap[gpIdKey] = bidderExt.GpId
	}

	imp.Ext = nil
	if len(extMap) > 0 {
		ext, err := json.Marshal(extMap)
		if err == nil {
			imp.Ext = ext
		}
	}

	return wrapExt, pubID, nil
}

// extractPubmaticExtFromRequest parse the req.ext to fetch wrapper and acat params
func extractPubmaticExtFromRequest(request *openrtb2.BidRequest) (extRequestAdServer, error) {
	// req.ext.prebid would always be there and Less nil cases to handle, more safe!
	var pmReqExt extRequestAdServer

	if request == nil || len(request.Ext) == 0 {
		return pmReqExt, nil
	}

	reqExt := &openrtb_ext.ExtRequest{}
	err := jsonutil.Unmarshal(request.Ext, &reqExt)
	if err != nil {
		return pmReqExt, fmt.Errorf("error decoding Request.ext : %s", err.Error())
	}

	reqExtBidderParams := make(map[string]json.RawMessage)
	if reqExt.Prebid.BidderParams != nil {
		err = jsonutil.Unmarshal(reqExt.Prebid.BidderParams, &reqExtBidderParams)
		if err != nil {
			return pmReqExt, err
		}
	}

	//get request ext bidder params
	if wrapperObj, present := reqExtBidderParams["wrapper"]; present && len(wrapperObj) != 0 {
		wrpExt := &pubmaticWrapperExt{}
		err = jsonutil.Unmarshal(wrapperObj, wrpExt)
		if err != nil {
			return pmReqExt, err
		}
		pmReqExt.Wrapper = wrpExt
	}

	if acatBytes, ok := reqExtBidderParams["acat"]; ok {
		var acat []string
		err = jsonutil.Unmarshal(acatBytes, &acat)
		if err != nil {
			return pmReqExt, err
		}
		for i := 0; i < len(acat); i++ {
			acat[i] = strings.TrimSpace(acat[i])
		}
		pmReqExt.Acat = acat
	}

	if allowedBidders := getAlternateBidderCodesFromRequestExt(reqExt); allowedBidders != nil {
		pmReqExt.Marketplace = &marketplaceReqExt{AllowedBidders: allowedBidders}
	}

	return pmReqExt, nil
}

func getAlternateBidderCodesFromRequestExt(reqExt *openrtb_ext.ExtRequest) []string {
	if reqExt == nil || reqExt.Prebid.AlternateBidderCodes == nil {
		return nil
	}

	allowedBidders := []string{"pubmatic"}
	if reqExt.Prebid.AlternateBidderCodes.Enabled {
		if pmABC, ok := reqExt.Prebid.AlternateBidderCodes.Bidders["pubmatic"]; ok && pmABC.Enabled {
			if pmABC.AllowedBidderCodes == nil || (len(pmABC.AllowedBidderCodes) == 1 && pmABC.AllowedBidderCodes[0] == "*") {
				return []string{"all"}
			}
			return append(allowedBidders, pmABC.AllowedBidderCodes...)
		}
	}

	return allowedBidders
}

func addKeywordsToExt(keywords []*openrtb_ext.ExtImpPubmaticKeyVal, extMap map[string]interface{}) {
	for _, keyVal := range keywords {
		if len(keyVal.Values) == 0 {
			continue
		} else {
			key := keyVal.Key
			if keyVal.Key == pmZoneIDKeyNameOld {
				key = pmZoneIDKeyName
			}
			extMap[key] = strings.Join(keyVal.Values[:], ",")
		}
	}
}

func (a *PubmaticAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			if len(bid.Cat) > 1 {
				bid.Cat = bid.Cat[0:1]
			}

			mType, err := getMediaTypeForBid(&bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			typedBid := &adapters.TypedBid{
				Bid:      &bid,
				BidVideo: &openrtb_ext.ExtBidPrebidVideo{},
				BidType:  mType,
			}

			var bidExt *pubmaticBidExt
			err = jsonutil.Unmarshal(bid.Ext, &bidExt)
			if err != nil {
				errs = append(errs, err)
			} else if bidExt != nil {
				typedBid.Seat = openrtb_ext.BidderName(bidExt.Marketplace)

				if bidExt.PrebidDealPriority > 0 {
					typedBid.DealPriority = bidExt.PrebidDealPriority
				}

				if bidExt.VideoCreativeInfo != nil && bidExt.VideoCreativeInfo.Duration != nil {
					typedBid.BidVideo.Duration = *bidExt.VideoCreativeInfo.Duration
				}

				typedBid.BidMeta = &openrtb_ext.ExtBidPrebidMeta{MediaType: string(mType)}
				if bidExt.InBannerVideo {
					typedBid.BidMeta.MediaType = string(openrtb_ext.BidTypeVideo)
				}
			}

			if mType == openrtb_ext.BidTypeNative {
				bid.AdM, err = getNativeAdm(bid.AdM)
				if err != nil {
					errs = append(errs, err)
				}
			}

			bidResponse.Bids = append(bidResponse.Bids, typedBid)
		}
	}
	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	}

	if bidResp.Ext != nil {
		var bidRespExt respExt
		if err := jsonutil.Unmarshal(bidResp.Ext, &bidRespExt); err == nil && bidRespExt.FledgeAuctionConfigs != nil {
			bidResponse.FledgeAuctionConfigs = make([]*openrtb_ext.FledgeAuctionConfig, 0, len(bidRespExt.FledgeAuctionConfigs))
			for impId, config := range bidRespExt.FledgeAuctionConfigs {
				fledgeAuctionConfig := &openrtb_ext.FledgeAuctionConfig{
					ImpId:  impId,
					Config: config,
				}
				bidResponse.FledgeAuctionConfigs = append(bidResponse.FledgeAuctionConfigs, fledgeAuctionConfig)
			}
		}
	}
	return bidResponse, errs
}

func getNativeAdm(adm string) (string, error) {
	var err error
	nativeAdm := make(map[string]interface{})
	err = jsonutil.Unmarshal([]byte(adm), &nativeAdm)
	if err != nil {
		return adm, errors.New("unable to unmarshal native adm")
	}

	// move bid.adm.native to bid.adm
	if _, ok := nativeAdm["native"]; ok {
		//using jsonparser to avoid marshaling, encode escape, etc.
		value, _, _, err := jsonparser.Get([]byte(adm), string(openrtb_ext.BidTypeNative))
		if err != nil {
			return adm, errors.New("unable to get native adm")
		}
		adm = string(value)
	}

	return adm, nil
}

// getMapFromJSON converts JSON to map
func getMapFromJSON(source json.RawMessage) map[string]interface{} {
	if source != nil {
		dataMap := make(map[string]interface{})
		err := jsonutil.Unmarshal(source, &dataMap)
		if err == nil {
			return dataMap
		}
	}
	return nil
}

// populateFirstPartyDataImpAttributes will parse imp.ext.data and populate imp extMap
func populateFirstPartyDataImpAttributes(data json.RawMessage, extMap map[string]interface{}) {

	dataMap := getMapFromJSON(data)

	if dataMap == nil {
		return
	}

	populateAdUnitKey(data, dataMap, extMap)
	populateDctrKey(dataMap, extMap)
}

// populateAdUnitKey parses data object to read and populate DFP adunit key
func populateAdUnitKey(data json.RawMessage, dataMap, extMap map[string]interface{}) {

	if name, err := jsonparser.GetString(data, "adserver", "name"); err == nil && name == AdServerGAM {
		if adslot, err := jsonparser.GetString(data, "adserver", "adslot"); err == nil && adslot != "" {
			extMap[ImpExtAdUnitKey] = adslot
		}
	}

	//imp.ext.dfp_ad_unit_code is not set, then check pbadslot in imp.ext.data
	if extMap[ImpExtAdUnitKey] == nil && dataMap[PBAdslotKey] != nil {
		extMap[ImpExtAdUnitKey] = dataMap[PBAdslotKey].(string)
	}
}

// populateDctrKey reads key-val pairs from imp.ext.data and add it in imp.ext.key_val
func populateDctrKey(dataMap, extMap map[string]interface{}) {
	var dctr strings.Builder

	//append dctr key if already present in extMap
	if extMap[dctrKeyName] != nil {
		dctr.WriteString(extMap[dctrKeyName].(string))
	}

	for key, val := range dataMap {

		//ignore 'pbaslot' and 'adserver' key as they are not targeting keys
		if key == PBAdslotKey || key == AdServerKey {
			continue
		}

		//separate key-val pairs in dctr string by pipe(|)
		if dctr.Len() > 0 {
			dctr.WriteString("|")
		}

		//trimming spaces from key
		key = strings.TrimSpace(key)

		switch typedValue := val.(type) {
		case string:
			if _, err := fmt.Fprintf(&dctr, "%s=%s", key, strings.TrimSpace(typedValue)); err != nil {
				continue
			}

		case float64, bool:
			if _, err := fmt.Fprintf(&dctr, "%s=%v", key, typedValue); err != nil {
				continue
			}

		case []interface{}:
			if valStrArr := getStringArray(typedValue); len(valStrArr) > 0 {
				valStr := strings.Join(valStrArr[:], ",")
				if _, err := fmt.Fprintf(&dctr, "%s=%s", key, valStr); err != nil {
					continue
				}
			}
		}
	}

	if dctrStr := dctr.String(); dctrStr != "" {
		extMap[dctrKeyName] = strings.TrimSuffix(dctrStr, "|")
	}
}

// getStringArray converts interface of type string array to string array
func getStringArray(array []interface{}) []string {
	aString := make([]string, len(array))
	for i, v := range array {
		if str, ok := v.(string); ok {
			aString[i] = strings.TrimSpace(str)
		} else {
			return nil
		}
	}
	return aString
}

// getMediaTypeForBid returns the Mtype
func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	// setting "banner" as the default bid type
	mType := openrtb_ext.BidTypeBanner
	if bid != nil {
		switch bid.MType {
		case openrtb2.MarkupBanner:
			mType = openrtb_ext.BidTypeBanner
		case openrtb2.MarkupVideo:
			mType = openrtb_ext.BidTypeVideo
		case openrtb2.MarkupAudio:
			mType = openrtb_ext.BidTypeAudio
		case openrtb2.MarkupNative:
			mType = openrtb_ext.BidTypeNative
		default:
			return "", &errortypes.BadServerResponse{
				Message: fmt.Sprintf("failed to parse bid mtype (%d) for impression id %s", bid.MType, bid.ImpID),
			}
		}
	}
	return mType, nil
}

// Builder builds a new instance of the Pubmatic adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &PubmaticAdapter{
		URI:        config.Endpoint,
		bidderName: string(bidderName),
	}
	return bidder, nil
}

// getDisplayManagerAndVer returns the display manager and version from the request.app.ext or request.app.prebid.ext source and version
func getDisplayManagerAndVer(app *openrtb2.App) (string, string) {
	if source, err := jsonparser.GetString(app.Ext, openrtb_ext.PrebidExtKey, "source"); err == nil && source != "" {
		if version, err := jsonparser.GetString(app.Ext, openrtb_ext.PrebidExtKey, "version"); err == nil && version != "" {
			return source, version
		}
	}

	if source, err := jsonparser.GetString(app.Ext, "source"); err == nil && source != "" {
		if version, err := jsonparser.GetString(app.Ext, "version"); err == nil && version != "" {
			return source, version
		}
	}
	return "", ""
}
