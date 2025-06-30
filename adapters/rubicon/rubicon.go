package rubicon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/prebid/prebid-server/v3/version"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/maputil"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
)

const badvLimitSize = 50

var bannerExtContent = []byte(`{"rp":{"mime":"text/html"}}`)

type RubiconAdapter struct {
	URI          string
	externalURI  string
	XAPIUsername string
	XAPIPassword string
}

type rubiconData struct {
	AdServer rubiconAdServer `json:"adserver"`
	PbAdSlot string          `json:"pbadslot"`
}

type rubiconAdServer struct {
	Name   string `json:"name"`
	AdSlot string `json:"adslot"`
}

type rubiconExtImpBidder struct {
	Prebid *openrtb_ext.ExtImpPrebid `json:"prebid"`
	Bidder openrtb_ext.ExtImpRubicon `json:"bidder"`
	Gpid   string                    `json:"gpid"`
	Skadn  json.RawMessage           `json:"skadn,omitempty"`
	Tid    string                    `json:"tid"`
	Data   json.RawMessage           `json:"data"`
}

type bidRequestExt struct {
	Prebid bidRequestExtPrebid `json:"prebid"`
}

type bidRequestExtPrebid struct {
	Bidders  bidRequestExtPrebidBidders `json:"bidders"`
	MultiBid []*openrtb_ext.ExtMultiBid `json:"multibid,omitempty"`
}

type bidRequestExtPrebidBidders struct {
	Rubicon prebidBiddersRubicon `json:"rubicon,omitempty"`
}

type prebidBiddersRubicon struct {
	Debug prebidBiddersRubiconDebug `json:"debug,omitempty"`
}

type prebidBiddersRubiconDebug struct {
	CpmOverride float64 `json:"cpmoverride,omitempty"`
}

type rubiconImpExtRPTrack struct {
	Mint        string `json:"mint"`
	MintVersion string `json:"mint_version"`
}

type rubiconImpExt struct {
	RP      rubiconImpExtRP `json:"rp,omitempty"`
	GPID    string          `json:"gpid,omitempty"`
	Skadn   json.RawMessage `json:"skadn,omitempty"`
	Tid     string          `json:"tid,omitempty"`
	MaxBids *int            `json:"maxbids,omitempty"`
}

type rubiconImpExtRP struct {
	ZoneID int                  `json:"zone_id"`
	Target json.RawMessage      `json:"target,omitempty"`
	Track  rubiconImpExtRPTrack `json:"track"`
}

type rubiconUserExtRP struct {
	Target json.RawMessage `json:"target,omitempty"`
}

type rubiconDataExt struct {
	SegTax int `json:"segtax"`
}

type rubiconUserExt struct {
	Eids    []openrtb2.EID   `json:"eids,omitempty"`
	RP      rubiconUserExtRP `json:"rp"`
	Data    json.RawMessage  `json:"data,omitempty"`
	Consent string           `json:"consent,omitempty"`
}

type rubiconSiteExtRP struct {
	SiteID int             `json:"site_id"`
	Target json.RawMessage `json:"target,omitempty"`
}

type rubiconSiteExt struct {
	RP rubiconSiteExtRP `json:"rp"`
}

type rubiconPubExtRP struct {
	AccountID int `json:"account_id"`
}

type rubiconPubExt struct {
	RP rubiconPubExtRP `json:"rp"`
}

type rubiconBannerExtRP struct {
	MIME string `json:"mime"`
}

type rubiconBannerExt struct {
	RP rubiconBannerExtRP `json:"rp"`
}

// ***** Video Extension *****
type rubiconVideoExt struct {
	Skip      int               `json:"skip,omitempty"`
	SkipDelay int               `json:"skipdelay,omitempty"`
	VideoType string            `json:"videotype,omitempty"`
	RP        rubiconVideoExtRP `json:"rp"`
}

type rubiconVideoExtRP struct {
	SizeID int `json:"size_id,omitempty"`
}

type rubiconDeviceExtRP struct {
	PixelRatio float64 `json:"pixelratio"`
}

type rubiconDeviceExt struct {
	RP rubiconDeviceExtRP `json:"rp"`
}

type rubiconBidResponse struct {
	openrtb2.BidResponse
	SeatBid []rubiconSeatBid `json:"seatbid,omitempty"`
}

type rubiconSeatBid struct {
	openrtb2.SeatBid
	Buyer string       `json:"buyer,omitempty"`
	Bid   []rubiconBid `json:"bid"`
}

type rubiconBid struct {
	openrtb2.Bid
	AdmNative json.RawMessage `json:"adm_native,omitempty"`
}

type extPrebid struct {
	Prebid *openrtb_ext.ExtBidPrebid `json:"prebid,omitempty"`
	Bidder json.RawMessage           `json:"bidder,omitempty"`
}

func appendTrackerToUrl(uri string, tracker string) (res string) {
	// Append integration method. Adapter init happens once
	urlObject, err := url.Parse(uri)
	// No other exception throwing mechanism in this stack, so ignoring parse errors.
	if err == nil {
		values := urlObject.Query()
		values.Add("tk_xint", tracker)
		urlObject.RawQuery = values.Encode()
		res = urlObject.String()
	} else {
		res = uri
	}
	return
}

// Builder builds a new instance of the Rubicon adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	uri := appendTrackerToUrl(config.Endpoint, config.XAPI.Tracker)

	bidder := &RubiconAdapter{
		URI:          uri,
		externalURI:  server.ExternalUrl,
		XAPIUsername: config.XAPI.Username,
		XAPIPassword: config.XAPI.Password,
	}
	return bidder, nil
}

func updateRequestTo26(r *openrtb2.BidRequest) error {
	if r.Regs != nil {
		regsCopy := *r.Regs
		r.Regs = &regsCopy
	}

	if r.Source != nil {
		sourceCopy := *r.Source
		r.Source = &sourceCopy
	}

	if r.User != nil {
		userCopy := *r.User
		r.User = &userCopy
	}

	requestWrapper := &openrtb_ext.RequestWrapper{BidRequest: r}

	if err := openrtb_ext.ConvertUpTo26(requestWrapper); err != nil {
		return err
	}

	return requestWrapper.RebuildRequest()
}

func (a *RubiconAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	err := updateRequestTo26(request)

	if err != nil {
		return nil, []error{err}
	}

	numRequests := len(request.Imp)
	requestData := make([]*adapters.RequestData, 0, numRequests)
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")

	impsToExtNotGrouped, errs := createImpsToExtMap(request.Imp)
	impsToExtMap := prepareImpsToExtMap(impsToExtNotGrouped)

	maxBids := getMaxBids(request)

	rubiconRequest := *request
	for imp, bidderExt := range impsToExtMap {
		rubiconExt := bidderExt.Bidder
		target, err := a.updateImpRpTarget(bidderExt, rubiconExt, *imp, request.Site, request.App)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		siteId, err := rubiconExt.SiteId.Int64()
		if err != nil {
			errs = append(errs, err)
			continue
		}

		zoneId, err := rubiconExt.ZoneId.Int64()
		if err != nil {
			errs = append(errs, err)
			continue
		}

		impExt := rubiconImpExt{
			RP: rubiconImpExtRP{
				ZoneID: int(zoneId),
				Target: target,
				Track:  rubiconImpExtRPTrack{Mint: "", MintVersion: ""},
			},
			GPID:    bidderExt.Gpid,
			Skadn:   bidderExt.Skadn,
			Tid:     bidderExt.Tid,
			MaxBids: maxBids,
		}

		imp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		secure := int8(1)
		imp.Secure = &secure

		resolvedBidFloor, err := resolveBidFloor(imp.BidFloor, imp.BidFloorCur, reqInfo)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("Unable to convert provided bid floor currency from %s to USD",
					imp.BidFloorCur),
			})
			continue
		}

		if resolvedBidFloor >= 0 {
			imp.BidFloor = resolvedBidFloor
			if imp.BidFloorCur != "" {
				imp.BidFloorCur = "USD"
			}
		}

		if request.User != nil {
			userCopy := *request.User
			target, err := updateUserRpTargetWithFpdAttributes(rubiconExt.Visitor, userCopy)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			userExtRP := rubiconUserExt{RP: rubiconUserExtRP{Target: target}}

			if len(userCopy.EIDs) > 0 {
				userExtRP.Eids = userCopy.EIDs
			}

			if userCopy.Consent != "" {
				userExtRP.Consent = userCopy.Consent
				userCopy.Consent = ""
			}

			userCopy.Ext, err = json.Marshal(&userExtRP)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			userCopy.Geo = nil
			userCopy.Yob = 0
			userCopy.Gender = ""
			userCopy.EIDs = nil

			rubiconRequest.User = &userCopy
		}

		if request.Device != nil {
			deviceCopy := *request.Device
			deviceExt := rubiconDeviceExt{RP: rubiconDeviceExtRP{PixelRatio: request.Device.PxRatio}}
			deviceCopy.Ext, err = json.Marshal(&deviceExt)
			rubiconRequest.Device = &deviceCopy
		}

		isVideo := isVideo(*imp)
		impType := openrtb_ext.BidTypeVideo
		requestNative := make(map[string]interface{})
		if isVideo {
			videoCopy := *imp.Video

			// if imp.rwdd = 1, set imp.video.ext.videotype = "rewarded"
			var videoType = ""
			if imp.Rwdd == 1 {
				videoType = "rewarded"
				imp.Rwdd = 0
			}
			videoExt := rubiconVideoExt{
				Skip:      rubiconExt.Video.Skip,
				SkipDelay: rubiconExt.Video.SkipDelay,
				VideoType: videoType,
				RP:        rubiconVideoExtRP{SizeID: rubiconExt.Video.VideoSizeID},
			}
			videoCopy.Ext, err = json.Marshal(&videoExt)
			imp.Video = &videoCopy
			imp.Banner = nil
			imp.Native = nil
		} else if imp.Banner != nil {
			bannerCopy := *imp.Banner
			if len(bannerCopy.Format) < 1 && (bannerCopy.W == nil || *bannerCopy.W == 0 && bannerCopy.H == nil || *bannerCopy.H == 0) {
				errs = append(errs, &errortypes.BadInput{
					Message: "rubicon imps must have at least one imp.format element",
				})
				continue
			}
			bannerCopy.Ext = bannerExtContent
			if err != nil {
				errs = append(errs, err)
				continue
			}
			imp.Banner = &bannerCopy
			imp.Video = nil
			imp.Native = nil
			impType = openrtb_ext.BidTypeBanner
		} else {
			native, err := resolveNativeObject(imp.Native, requestNative)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			imp.Native = native
			imp.Video = nil
			impType = openrtb_ext.BidTypeNative
		}

		accountId, err := rubiconExt.AccountId.Int64()
		if err != nil {
			errs = append(errs, err)
			continue
		}

		pubExt := rubiconPubExt{RP: rubiconPubExtRP{AccountID: int(accountId)}}

		if request.Site != nil {
			siteCopy := *request.Site
			siteExtRP := rubiconSiteExt{RP: rubiconSiteExtRP{SiteID: int(siteId)}}
			if siteCopy.Content != nil {
				siteTarget := make(map[string]interface{})
				updateExtWithIabAttribute(siteTarget, siteCopy.Content.Data, []int{1, 2, 5, 6})
				if len(siteTarget) > 0 {
					updatedSiteTarget, err := json.Marshal(siteTarget)
					if err != nil {
						errs = append(errs, err)
						continue
					}
					siteExtRP.RP.Target = updatedSiteTarget
				}
			}

			siteCopy.Ext, err = json.Marshal(&siteExtRP)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			siteCopy.Publisher = &openrtb2.Publisher{}
			siteCopy.Publisher.Ext, err = json.Marshal(&pubExt)
			rubiconRequest.Site = &siteCopy
		} else {
			appCopy := *request.App
			appCopy.Ext, err = json.Marshal(rubiconSiteExt{RP: rubiconSiteExtRP{SiteID: int(siteId)}})
			if err != nil {
				errs = append(errs, &errortypes.BadInput{Message: err.Error()})
			}
			appCopy.Publisher = &openrtb2.Publisher{}
			appCopy.Publisher.Ext, err = json.Marshal(&pubExt)
			if err != nil {
				errs = append(errs, &errortypes.BadInput{Message: err.Error()})
			}
			rubiconRequest.App = &appCopy
		}

		if request.Source != nil && request.Source.SChain != nil {
			sourceCopy := *request.Source

			var sourceCopyExt openrtb_ext.ExtSource
			if sourceCopy.Ext != nil {
				if err = jsonutil.Unmarshal(sourceCopy.Ext, &sourceCopyExt); err != nil {
					errs = append(errs, &errortypes.BadInput{Message: err.Error()})
					continue
				}
			} else {
				sourceCopyExt = openrtb_ext.ExtSource{}
			}

			sourceCopyExt.SChain = sourceCopy.SChain
			sourceCopy.SChain = nil

			sourceCopy.Ext, err = json.Marshal(&sourceCopyExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			rubiconRequest.Source = &sourceCopy
		}

		if request.Regs != nil && (request.Regs.GDPR != nil || request.Regs.USPrivacy != "") {
			regsCopy := *request.Regs

			var regsCopyExt openrtb_ext.ExtRegs
			if regsCopy.Ext != nil {
				if err = jsonutil.Unmarshal(regsCopy.Ext, &regsCopyExt); err != nil {
					errs = append(errs, &errortypes.BadInput{Message: err.Error()})
					continue
				}
			} else {
				regsCopyExt = openrtb_ext.ExtRegs{}
			}

			if regsCopy.GDPR != nil {
				regsCopyExt.GDPR = regsCopy.GDPR
			}
			if regsCopy.USPrivacy != "" {
				regsCopyExt.USPrivacy = regsCopy.USPrivacy
			}

			regsCopy.Ext, err = json.Marshal(&regsCopyExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			regsCopy.GDPR = nil
			regsCopy.USPrivacy = ""

			rubiconRequest.Regs = &regsCopy
		}

		reqBadv := request.BAdv
		if reqBadv != nil {
			if len(reqBadv) > badvLimitSize {
				rubiconRequest.BAdv = reqBadv[:badvLimitSize]
			}
		}

		rubiconRequest.Imp = []openrtb2.Imp{*imp}
		rubiconRequest.Cur = nil
		rubiconRequest.Ext = nil

		reqJSON, err := json.Marshal(rubiconRequest)
		if impType == openrtb_ext.BidTypeNative && len(requestNative) > 0 {
			reqJSON, err = setImpNative(reqJSON, requestNative)
		}

		if err != nil {
			errs = append(errs, err)
			continue
		}

		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     a.URI,
			Body:    reqJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(rubiconRequest.Imp),
		}
		reqData.SetBasicAuth(a.XAPIUsername, a.XAPIPassword)
		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

func getMaxBids(bidRequest *openrtb2.BidRequest) *int {
	var bidRequestExt bidRequestExt
	if err := jsonutil.Unmarshal(bidRequest.Ext, &bidRequestExt); err != nil {
		return nil
	}

	if len(bidRequestExt.Prebid.MultiBid) == 0 {
		return nil
	}

	multiBid := bidRequestExt.Prebid.MultiBid[0]

	if multiBid == nil {
		return nil
	}

	return multiBid.MaxBids
}

func createImpsToExtMap(imps []openrtb2.Imp) (map[*openrtb2.Imp]rubiconExtImpBidder, []error) {
	impsToExtMap := make(map[*openrtb2.Imp]rubiconExtImpBidder)
	errs := make([]error, 0)
	var err error
	for _, imp := range imps {
		impCopy := imp
		var bidderExt rubiconExtImpBidder
		if err = jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}
		impsToExtMap[&impCopy] = bidderExt
	}

	return impsToExtMap, errs
}

func prepareImpsToExtMap(impsToExtMap map[*openrtb2.Imp]rubiconExtImpBidder) map[*openrtb2.Imp]rubiconExtImpBidder {
	preparedImpsToExtMap := make(map[*openrtb2.Imp]rubiconExtImpBidder)
	for imp, bidderExt := range impsToExtMap {
		if bidderExt.Bidder.BidOnMultiformat == false { //nolint: gosimple,staticcheck
			impCopy := imp
			preparedImpsToExtMap[impCopy] = bidderExt
			continue
		}

		splitImps := splitMultiFormatImp(imp)
		for _, imp := range splitImps {
			impCopy := imp
			preparedImpsToExtMap[impCopy] = bidderExt
		}
	}

	return preparedImpsToExtMap
}

func splitMultiFormatImp(imp *openrtb2.Imp) []*openrtb2.Imp {
	splitImps := make([]*openrtb2.Imp, 0)
	if imp.Banner != nil {
		impCopy := *imp
		impCopy.Video = nil
		impCopy.Native = nil
		impCopy.Audio = nil
		splitImps = append(splitImps, &impCopy)
	}

	if imp.Video != nil {
		impCopy := *imp
		impCopy.Banner = nil
		impCopy.Native = nil
		impCopy.Audio = nil
		splitImps = append(splitImps, &impCopy)
	}

	if imp.Native != nil {
		impCopy := *imp
		impCopy.Banner = nil
		impCopy.Video = nil
		impCopy.Audio = nil
		splitImps = append(splitImps, &impCopy)
	}

	if imp.Audio != nil {
		impCopy := *imp
		impCopy.Banner = nil
		impCopy.Video = nil
		impCopy.Native = nil
		splitImps = append(splitImps, &impCopy)
	}

	return splitImps
}

func resolveBidFloor(bidFloor float64, bidFloorCur string, reqInfo *adapters.ExtraRequestInfo) (float64, error) {
	if bidFloor > 0 && bidFloorCur != "" && strings.ToUpper(bidFloorCur) != "USD" {
		return reqInfo.ConvertCurrency(bidFloor, bidFloorCur, "USD")
	}

	return bidFloor, nil
}

func (a *RubiconAdapter) updateImpRpTarget(extImp rubiconExtImpBidder, extImpRubicon openrtb_ext.ExtImpRubicon,
	imp openrtb2.Imp, site *openrtb2.Site, app *openrtb2.App) (json.RawMessage, error) {

	existingTarget, _, _, err := jsonparser.Get(imp.Ext, "rp", "target")
	if isNotKeyPathError(err) {
		return nil, err
	}
	target, err := rawJSONToMap(existingTarget)
	if err != nil {
		return nil, err
	}
	err = populateFirstPartyDataAttributes(extImpRubicon.Inventory, target)
	if err != nil {
		return nil, err
	}

	if site != nil {
		siteExtData, _, _, err := jsonparser.Get(site.Ext, "data")
		if isNotKeyPathError(err) {
			return nil, err
		}
		err = populateFirstPartyDataAttributes(siteExtData, target)
		if err != nil {
			return nil, err
		}
		if site.Page != "" {
			addStringAttribute(site.Page, target, "page")
		}
	} else {
		appExtData, _, _, err := jsonparser.Get(app.Ext, "data")
		if isNotKeyPathError(err) {
			return nil, err
		}
		err = populateFirstPartyDataAttributes(appExtData, target)
		if err != nil {
			return nil, err
		}
	}

	if len(extImp.Data) > 0 {
		err = populateFirstPartyDataAttributes(extImp.Data, target)
	}
	if isNotKeyPathError(err) {
		return nil, err
	}

	var data rubiconData
	if len(extImp.Data) > 0 {
		err := jsonutil.Unmarshal(extImp.Data, &data)
		if err != nil {
			return nil, err
		}
	}

	if data.PbAdSlot != "" {
		target["pbadslot"] = data.PbAdSlot
	} else {
		dfpAdUnitCode := extractDfpAdUnitCode(data)
		if dfpAdUnitCode != "" {
			target["dfp_ad_unit_code"] = dfpAdUnitCode
		}
	}

	if len(extImpRubicon.Keywords) > 0 {
		addStringArrayAttribute(extImpRubicon.Keywords, target, "keywords")
	}

	target["pbs_login"] = a.XAPIUsername
	target["pbs_version"] = version.Ver
	target["pbs_url"] = a.externalURI

	updatedTarget, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}
	return updatedTarget, nil
}

func extractDfpAdUnitCode(data rubiconData) string {
	if data.AdServer.Name == "gam" && len(data.AdServer.AdSlot) != 0 {
		return data.AdServer.AdSlot
	}

	return ""
}

func isNotKeyPathError(err error) bool {
	return err != nil && err != jsonparser.KeyPathNotFoundError
}

func addStringAttribute(attribute string, target map[string]interface{}, attributeName string) {
	target[attributeName] = [1]string{attribute}
}

func addStringArrayAttribute(attribute []string, target map[string]interface{}, attributeName string) {
	target[attributeName] = attribute
}

func updateUserRpTargetWithFpdAttributes(visitor json.RawMessage, user openrtb2.User) (json.RawMessage, error) {
	existingTarget, _, _, err := jsonparser.Get(user.Ext, "rp", "target")
	if isNotKeyPathError(err) {
		return nil, err
	}
	target, err := rawJSONToMap(existingTarget)
	if err != nil {
		return nil, err
	}
	err = populateFirstPartyDataAttributes(visitor, target)
	if err != nil {
		return nil, err
	}
	userExtData, _, _, err := jsonparser.Get(user.Ext, "data")
	if isNotKeyPathError(err) {
		return nil, err
	}
	err = populateFirstPartyDataAttributes(userExtData, target)
	if err != nil {
		return nil, err
	}
	updateExtWithIabAttribute(target, user.Data, []int{4})

	updatedTarget, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}
	return updatedTarget, nil
}

func updateExtWithIabAttribute(target map[string]interface{}, data []openrtb2.Data, segTaxes []int) {
	var segmentIdsToCopy = getSegmentIdsToCopy(data, segTaxes)
	if len(segmentIdsToCopy) == 0 {
		return
	}

	target["iab"] = segmentIdsToCopy
}

func populateFirstPartyDataAttributes(source json.RawMessage, target map[string]interface{}) error {
	sourceAsMap, err := rawJSONToMap(source)
	if err != nil {
		return err
	}

	for key, val := range sourceAsMap {
		switch typedValue := val.(type) {
		case string:
			target[key] = [1]string{typedValue}
		case float64:
			if typedValue == float64(int(typedValue)) {
				target[key] = [1]string{strconv.Itoa(int(typedValue))}
			}
		case bool:
			target[key] = [1]string{strconv.FormatBool(typedValue)}
		case []interface{}:
			if isStringArray(typedValue) {
				target[key] = typedValue
			}
			if isBoolArray(typedValue) {
				target[key] = convertToStringArray(typedValue)
			}
		}
	}
	return nil
}

func isStringArray(array []interface{}) bool {
	for _, val := range array {
		if _, ok := val.(string); !ok {
			return false
		}
	}

	return true
}

func isBoolArray(array []interface{}) bool {
	for _, val := range array {
		if _, ok := val.(bool); !ok {
			return false
		}
	}

	return true
}

func convertToStringArray(arr []interface{}) []string {
	var stringArray []string
	for _, val := range arr {
		if boolVal, ok := val.(bool); ok {
			stringArray = append(stringArray, strconv.FormatBool(boolVal))
		}
	}

	return stringArray
}

func rawJSONToMap(message json.RawMessage) (map[string]interface{}, error) {
	if message == nil {
		return make(map[string]interface{}), nil
	}

	return mapFromRawJSON(message)
}

func mapFromRawJSON(message json.RawMessage) (map[string]interface{}, error) {
	targetAsMap := make(map[string]interface{})
	err := jsonutil.Unmarshal(message, &targetAsMap)
	if err != nil {
		return nil, err
	}
	return targetAsMap, nil
}

func getSegmentIdsToCopy(data []openrtb2.Data, segTaxValues []int) []string {
	var segmentIdsToCopy = make([]string, 0, len(data))

	for _, dataRecord := range data {
		if dataRecord.Ext != nil {
			var dataExtObject rubiconDataExt
			err := jsonutil.Unmarshal(dataRecord.Ext, &dataExtObject)
			if err != nil {
				continue
			}
			if contains(segTaxValues, dataExtObject.SegTax) {
				for _, segment := range dataRecord.Segment {
					segmentIdsToCopy = append(segmentIdsToCopy, segment.ID)
				}
			}
		}
	}
	return segmentIdsToCopy
}

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func isVideo(imp openrtb2.Imp) bool {
	video := imp.Video
	if video != nil {
		// Do any other media types exist? Or check required video fields.
		return imp.Banner == nil || isFullyPopulatedVideo(video)
	}
	return false
}

func isFullyPopulatedVideo(video *openrtb2.Video) bool {
	// These are just recommended video fields for XAPI
	return video.MIMEs != nil && video.Protocols != nil && video.MaxDuration != 0 && video.Linearity != 0
}

func resolveNativeObject(native *openrtb2.Native, target map[string]interface{}) (*openrtb2.Native, error) {
	if native == nil {
		return nil, fmt.Errorf("Native object is not present for request")
	}
	ver := native.Ver
	if ver == "1.0" || ver == "1.1" {
		return native, nil
	}

	err := jsonutil.Unmarshal([]byte(native.Request), &target)
	if err != nil {
		return nil, err
	}

	if _, ok := target["eventtrackers"].([]interface{}); !ok {
		return nil, fmt.Errorf("Eventtrackers are not present or not of array type")
	}

	context := target["context"]
	if context != nil {
		if _, ok := context.(float64); !ok {
			return nil, fmt.Errorf("Context is not of int type")
		}
	}

	if _, ok := target["plcmttype"].(float64); !ok {
		return nil, fmt.Errorf("Plcmttype is not present or not of int type")
	}

	return native, nil
}

func setImpNative(jsonData []byte, requestNative map[string]interface{}) ([]byte, error) {
	var jsonMap map[string]interface{}
	if err := jsonutil.Unmarshal(jsonData, &jsonMap); err != nil {
		return jsonData, err
	}

	var impMap map[string]interface{}
	if impSlice, ok := maputil.ReadEmbeddedSlice(jsonMap, "imp"); !ok {
		return jsonData, fmt.Errorf("unable to find imp in json data")
	} else if len(impSlice) == 0 {
		return jsonData, fmt.Errorf("unable to find imp[0] in json data")
	} else if impMap, ok = impSlice[0].(map[string]interface{}); !ok {
		return jsonData, fmt.Errorf("unexpected type for imp[0] found in json data")
	}

	nativeMap, ok := maputil.ReadEmbeddedMap(impMap, "native")
	if !ok {
		return jsonData, fmt.Errorf("unable to find imp[0].native in json data")
	}

	nativeMap["request_native"] = requestNative

	if jsonReEncoded, err := json.Marshal(jsonMap); err == nil {
		return jsonReEncoded, nil
	} else {
		return nil, fmt.Errorf("unable to encode json data (%v)", err)
	}
}

func (a *RubiconAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp rubiconBidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	var bidReq openrtb2.BidRequest
	if err := jsonutil.Unmarshal(externalRequest.Body, &bidReq); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	bidType := openrtb_ext.BidTypeNative

	isVideo := isVideo(bidReq.Imp[0])
	if isVideo {
		bidType = openrtb_ext.BidTypeVideo
	} else if bidReq.Imp[0].Banner != nil {
		bidType = openrtb_ext.BidTypeBanner
	}

	impToCpmOverride := mapImpIdToCpmOverride(internalRequest.Imp)
	cmpOverride := cmpOverrideFromBidRequest(internalRequest)

	for _, sb := range bidResp.SeatBid {
		buyer, err := strconv.Atoi(sb.Buyer)
		if err != nil {
			buyer = 0
		}
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]

			updatedBidExt := updateBidExtWithMeta(bid, buyer, sb.Seat)
			if updatedBidExt != nil {
				bid.Ext = updatedBidExt
			}
			bidCmpOverride, ok := impToCpmOverride[bid.ImpID]
			if !ok || bidCmpOverride == 0 {
				bidCmpOverride = cmpOverride
			}

			if bidCmpOverride > 0 {
				bid.Price = bidCmpOverride
			}

			if bid.Price != 0 {
				// Since Rubicon XAPI returns only one bid per response
				// copy response.bidid to openrtb_response.seatbid.bid.bidid
				if bid.ID == "0" {
					bid.ID = bidResp.BidID
				}

				resolvedAdm := resolveAdm(bid)
				if len(resolvedAdm) > 0 {
					bid.AdM = resolvedAdm
				}

				var ortbBid openrtb2.Bid // `targetStruct` can be anything of your choice

				rubiconBidAsBytes, _ := json.Marshal(bid)
				if len(rubiconBidAsBytes) > 0 {
					err = jsonutil.Unmarshal(rubiconBidAsBytes, &ortbBid)
					if err != nil {
						return nil, []error{err}
					}
				}

				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &ortbBid,
					BidType: bidType,
				})
			}
		}
	}
	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	}

	return bidResponse, nil
}

func mapImpIdToCpmOverride(imps []openrtb2.Imp) map[string]float64 {
	impIdToCmpOverride := make(map[string]float64)
	for _, imp := range imps {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			continue
		}

		var rubiconExt openrtb_ext.ExtImpRubicon
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &rubiconExt); err != nil {
			continue
		}

		impIdToCmpOverride[imp.ID] = rubiconExt.Debug.CpmOverride
	}
	return impIdToCmpOverride
}

func resolveAdm(bid rubiconBid) string {
	var bidAdm = bid.AdM
	if len(bidAdm) > 0 {
		return bidAdm
	}

	admObject := bid.AdmNative
	admObjectAsBytes, err := json.Marshal(&admObject)
	if err != nil {
		return ""
	}

	return string(admObjectAsBytes)
}

func cmpOverrideFromBidRequest(bidRequest *openrtb2.BidRequest) float64 {
	var bidRequestExt bidRequestExt
	if err := jsonutil.Unmarshal(bidRequest.Ext, &bidRequestExt); err != nil {
		return 0
	}

	return bidRequestExt.Prebid.Bidders.Rubicon.Debug.CpmOverride
}

func updateBidExtWithMeta(bid rubiconBid, buyer int, seat string) json.RawMessage {
	if buyer <= 0 && seat == "" {
		return nil
	}
	var bidExt *extPrebid
	if bid.Ext != nil {
		if err := jsonutil.Unmarshal(bid.Ext, &bidExt); err != nil {
			return nil
		}
	}

	if bidExt != nil {
		if bidExt.Prebid != nil {
			if bidExt.Prebid.Meta != nil {
				bidExt.Prebid.Meta.NetworkID = buyer
				bidExt.Prebid.Meta.Seat = seat
			} else {
				bidExt.Prebid.Meta = &openrtb_ext.ExtBidPrebidMeta{NetworkID: buyer, Seat: seat}
			}
		} else {
			bidExt.Prebid = &openrtb_ext.ExtBidPrebid{Meta: &openrtb_ext.ExtBidPrebidMeta{NetworkID: buyer, Seat: seat}}
		}
	} else {
		bidExt = &extPrebid{Prebid: &openrtb_ext.ExtBidPrebid{Meta: &openrtb_ext.ExtBidPrebidMeta{NetworkID: buyer, Seat: seat}}}
	}

	marshalledExt, err := json.Marshal(&bidExt)
	if err == nil {
		return marshalledExt
	}
	return nil
}
