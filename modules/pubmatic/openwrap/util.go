package openwrap

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"

	"slices"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	gppConstants "github.com/prebid/go-gpp/constants"
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/adapters"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/adunitconfig"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/nbr"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/profilemetadata"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/sdkutils"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	gppPolicy "github.com/prebid/prebid-server/v3/privacy/gpp"
)

var uidRegexp = regexp.MustCompile(`^(UID2|ID5|BGID|euid|PAIRID|IDL|connectid|firstid|utiq):`)

var (
	widthRegEx  *regexp.Regexp
	heightRegEx *regexp.Regexp
	auIDRegEx   *regexp.Regexp
	divRegEx    *regexp.Regexp

	openRTBDeviceOsAndroidRegex *regexp.Regexp
	androidUARegex              *regexp.Regexp
	iosUARegex                  *regexp.Regexp
	openRTBDeviceOsIosRegex     *regexp.Regexp
	mobileDeviceUARegex         *regexp.Regexp
	ctvRegex                    *regexp.Regexp
	accountIdSearchPath         = [...]struct {
		isApp  bool
		isDOOH bool
		key    []string
	}{
		{true, false, []string{"app", "publisher", "ext", openrtb_ext.PrebidExtKey, "parentAccount"}},
		{true, false, []string{"app", "publisher", "id"}},
		{false, false, []string{"site", "publisher", "ext", openrtb_ext.PrebidExtKey, "parentAccount"}},
		{false, false, []string{"site", "publisher", "id"}},
		{false, true, []string{"dooh", "publisher", "ext", openrtb_ext.PrebidExtKey, "parentAccount"}},
		{false, true, []string{"dooh", "publisher", "id"}},
	}
)

const (
	test = "_test"
)

var (
	protocols = []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST30, adcom1.CreativeVAST30Wrapper, adcom1.CreativeVAST40, adcom1.CreativeVAST40Wrapper}
)

func init() {
	widthRegEx = regexp.MustCompile(models.MACRO_WIDTH)
	heightRegEx = regexp.MustCompile(models.MACRO_HEIGHT)
	auIDRegEx = regexp.MustCompile(models.MACRO_AD_UNIT_ID)
	//auIndexRegEx := regexp.MustCompile(models.MACRO_AD_UNIT_INDEX)
	//integerRegEx := regexp.MustCompile(models.MACRO_INTEGER)
	divRegEx = regexp.MustCompile(models.MACRO_DIV)

	openRTBDeviceOsAndroidRegex = regexp.MustCompile(models.OpenRTBDeviceOsAndroidRegexPattern)
	androidUARegex = regexp.MustCompile(models.AndroidUARegexPattern)
	iosUARegex = regexp.MustCompile(models.IosUARegexPattern)
	openRTBDeviceOsIosRegex = regexp.MustCompile(models.OpenRTBDeviceOsIosRegexPattern)
	mobileDeviceUARegex = regexp.MustCompile(models.MobileDeviceUARegexPattern)
	ctvRegex = regexp.MustCompile(models.ConnectedDeviceUARegexPattern)
}

//	rCtx.DevicePlatform = getDevicePlatform(rCtx.UA, payload.BidRequest, rCtx.Platform, rCtx.PubIDStr, m.metricEngine)
//
// getDevicePlatform determines the device from which request has been generated
func getDevicePlatform(rCtx models.RequestCtx, bidRequest *openrtb2.BidRequest) models.DevicePlatform {
	userAgentString := rCtx.DeviceCtx.UA

	switch rCtx.Platform {
	case models.PLATFORM_AMP:
		return models.DevicePlatformMobileWeb

	case models.PLATFORM_APP:
		//Its mobile; now determine ios or android
		var os = ""
		if bidRequest != nil && bidRequest.Device != nil && len(bidRequest.Device.OS) != 0 {
			os = bidRequest.Device.OS
		}
		if platform := getMobileAppPlatform(os, userAgentString); platform != models.DevicePlatformNotDefined {
			return platform
		}

	case models.PLATFORM_DISPLAY:
		//Its web; now determine mobile or desktop
		var deviceType adcom1.DeviceType
		if bidRequest != nil && bidRequest.Device != nil && bidRequest.Device.DeviceType != 0 {
			deviceType = bidRequest.Device.DeviceType
		}
		if isMobile(deviceType, userAgentString) {
			return models.DevicePlatformMobileWeb
		}
		return models.DevicePlatformDesktop

	case models.PLATFORM_VIDEO:
		var deviceType adcom1.DeviceType
		if bidRequest != nil && bidRequest.Device != nil && bidRequest.Device.DeviceType != 0 {
			deviceType = bidRequest.Device.DeviceType
		}
		isCtv := isCTV(userAgentString)

		if deviceType != 0 {
			if deviceType == adcom1.DeviceTV || deviceType == adcom1.DeviceConnected || deviceType == adcom1.DeviceSetTopBox {
				return models.DevicePlatformConnectedTv
			}
		}

		if deviceType == 0 && isCtv {
			return models.DevicePlatformConnectedTv
		}

		if bidRequest != nil && bidRequest.Site != nil {
			//Its web; now determine mobile or desktop
			var deviceType adcom1.DeviceType
			if bidRequest.Device != nil {
				deviceType = bidRequest.Device.DeviceType
			}
			if isMobile(deviceType, userAgentString) {
				return models.DevicePlatformMobileWeb
			}
			return models.DevicePlatformDesktop
		}

		if bidRequest != nil && bidRequest.App != nil {
			//Its mobile; now determine ios or android
			var os = ""
			if bidRequest.Device != nil && len(bidRequest.Device.OS) != 0 {
				os = bidRequest.Device.OS
			}
			if isIos(os, userAgentString) {
				return models.DevicePlatformMobileAppIos
			} else if isAndroid(os, userAgentString) {
				return models.DevicePlatformMobileAppAndroid
			}
		}

	default:
		return models.DevicePlatformNotDefined

	}

	return models.DevicePlatformNotDefined
}

func isMobile(deviceType adcom1.DeviceType, userAgentString string) bool {
	if deviceType != 0 {
		return deviceType == adcom1.DeviceMobile || deviceType == adcom1.DeviceTablet || deviceType == adcom1.DevicePhone
	}

	if mobileDeviceUARegex.Match([]byte(strings.ToLower(userAgentString))) {
		return true
	}
	return false
}

func isIos(os string, userAgentString string) bool {
	if openRTBDeviceOsIosRegex.Match([]byte(strings.ToLower(os))) || iosUARegex.Match([]byte(strings.ToLower(userAgentString))) {
		return true
	}
	return false
}

func isAndroid(os string, userAgentString string) bool {
	if openRTBDeviceOsAndroidRegex.Match([]byte(strings.ToLower(os))) || androidUARegex.Match([]byte(strings.ToLower(userAgentString))) {
		return true
	}
	return false
}

// getMobileAppPlatform returns iOS or Android for mobile app traffic.
// It matches device.os with strings.EqualFold against "ios" or "android" only; otherwise it uses iosUARegex / androidUARegex on the user agent (same patterns as isIos/isAndroid UA checks).
func getMobileAppPlatform(os, userAgentString string) models.DevicePlatform {
	if strings.EqualFold(os, "ios") {
		return models.DevicePlatformMobileAppIos
	}
	if strings.EqualFold(os, "android") {
		return models.DevicePlatformMobileAppAndroid
	}

	if iosUARegex.Match([]byte(strings.ToLower(userAgentString))) {
		return models.DevicePlatformMobileAppIos
	}
	if androidUARegex.Match([]byte(strings.ToLower(userAgentString))) {
		return models.DevicePlatformMobileAppAndroid
	}
	return models.DevicePlatformNotDefined
}

// GetIntArray converts interface to int array if it is compatible
func GetIntArray(val interface{}) []int {
	intArray := make([]int, 0)
	valArray, ok := val.([]interface{})
	if !ok {
		return nil
	}
	for _, x := range valArray {
		var intVal int
		intVal = GetInt(x)
		intArray = append(intArray, intVal)
	}
	return intArray
}

// GetInt converts interface to int if it is compatible
func GetInt(val interface{}) int {
	var result int
	if val != nil {
		switch val.(type) {
		case int:
			result = val.(int)
		case float64:
			val := val.(float64)
			result = int(val)
		case float32:
			val := val.(float32)
			result = int(val)
		}
	}
	return result
}

func getSourceAndOrigin(bidRequest *openrtb2.BidRequest) (string, string) {
	var source, origin string
	if bidRequest.Site != nil {
		if len(bidRequest.Site.Domain) != 0 {
			source = bidRequest.Site.Domain
			origin = source
		} else if len(bidRequest.Site.Page) != 0 {
			source = getDomainFromUrl(bidRequest.Site.Page)
			origin = source

		}
	} else if bidRequest.App != nil {
		source = bidRequest.App.Bundle
		origin = source
	}
	return source, origin
}

// getHostName Generates server name from node and pod name in K8S  environment
func GetHostName() string {

	var nodeName string
	if nodeName, _ = os.LookupEnv(models.ENV_VAR_NODE_NAME); nodeName == "" {
		nodeName = models.DEFAULT_NODENAME
	} else {
		nodeName = strings.Split(nodeName, ".")[0]
	}

	podName := getPodName()

	return nodeName + ":" + podName
}

func getPodName() string {

	var podName string
	if podName, _ = os.LookupEnv(models.ENV_VAR_POD_NAME); podName == "" {
		podName = models.DEFAULT_PODNAME
	} else {
		podName = strings.TrimPrefix(podName, "ssheaderbidding-")
	}
	return podName
}

// RecordPublisherPartnerNoCookieStats parse request cookies and records the stats if cookie is not found for partner
func RecordPublisherPartnerNoCookieStats(rctx models.RequestCtx) {

	for _, partnerConfig := range rctx.PartnerConfigMap {
		if partnerConfig[models.SERVER_SIDE_FLAG] == "0" {
			continue
		}

		partnerName := partnerConfig[models.PREBID_PARTNER_NAME]
		syncer := models.SyncerMap[adapters.ResolveOWBidder(partnerName)]
		if syncer != nil {
			uid, _, _ := rctx.ParsedUidCookie.GetUID(syncer.Key())
			if uid != "" {
				continue
			}
		}
		rctx.MetricsEngine.RecordPublisherPartnerNoCookieStats(rctx.PubIDStr, partnerConfig[models.BidderCode])
	}
}

// getPubmaticErrorCode is temporary function which returns the pubmatic specific error code for standardNBR code
func getPubmaticErrorCode(standardNBR openrtb3.NoBidReason) int {
	switch standardNBR {
	case nbr.InvalidPublisherID:
		return 604 // ErrMissingPublisherID

	case nbr.InvalidRequestExt, openrtb3.NoBidInvalidRequest:
		return 18 // ErrBadRequest

	case nbr.InvalidProfileID:
		return 700 // ErrMissingProfileID

	case nbr.AllPartnerThrottled:
		return 11 // ErrAllPartnerThrottled

	case nbr.InvalidPriceGranularityConfig:
		return 26 // ErrPrebidInvalidCustomPriceGranularity

	case nbr.InvalidImpressionTagID:
		return 605 // ErrMissingTagID

	case nbr.InvalidProfileConfiguration, nbr.InvalidPlatform, nbr.AllSlotsDisabled, nbr.ServerSidePartnerNotConfigured:
		return 6 // ErrInvalidConfiguration

	case nbr.InternalError:
		return 17 // ErrInvalidImpression

	case nbr.AllPartnersFiltered:
		return 26

	case nbr.RequestBlockedGeoFiltered:
		return int(nbr.RequestBlockedGeoFiltered)

	case nbr.APSSlotUUIDNotMapped:
		return 18 // ErrBadRequest — APS slot UUID not mapped to OW ad unit / profile
	}

	return -1
}

func isCTV(userAgent string) bool {
	return ctvRegex.Match([]byte(userAgent))
}

// getUserAgent returns value of bidRequest.Device.UA if present else returns empty string
func getUserAgent(bidRequest *openrtb2.BidRequest, defaultUA string) string {
	userAgent := defaultUA
	if bidRequest != nil && bidRequest.Device != nil && len(bidRequest.Device.UA) > 0 {
		userAgent = bidRequest.Device.UA
	}
	return userAgent
}

func getIP(bidRequest *openrtb2.BidRequest, defaultIP string) string {
	ip := defaultIP
	if bidRequest != nil && bidRequest.Device != nil {
		if len(bidRequest.Device.IP) > 0 {
			ip = bidRequest.Device.IP
		} else if len(bidRequest.Device.IPv6) > 0 {
			ip = bidRequest.Device.IPv6
		}
	}
	return ip
}

func getCountry(bidRequest *openrtb2.BidRequest) string {
	if bidRequest.Device != nil && bidRequest.Device.Geo != nil && bidRequest.Device.Geo.Country != "" {
		return bidRequest.Device.Geo.Country
	}
	if bidRequest.User != nil && bidRequest.User.Geo != nil && bidRequest.User.Geo.Country != "" {
		return bidRequest.User.Geo.Country
	}
	return ""
}

func getPlatformFromRequest(request *openrtb2.BidRequest) string {
	var platform string
	if request == nil {
		return platform
	}
	if request.Site != nil {
		return models.PLATFORM_DISPLAY
	}
	if request.App != nil {
		return models.PLATFORM_APP
	}
	return platform
}

// for AMP requests based on traffic percentage, we will decide to send video or not
// if traffic percentage is not defined then send video
// if traffic percentage is defined then send video based on percentage
func isVideoEnabledForAMP(adUnitConfig *adunitconfig.AdConfig) bool {
	if adUnitConfig == nil || adUnitConfig.Video == nil || adUnitConfig.Video.Enabled == nil || !*adUnitConfig.Video.Enabled {
		return false
	} else if adUnitConfig.Video.AmpTrafficPercentage == nil || rand.Intn(100) < *adUnitConfig.Video.AmpTrafficPercentage {
		return true
	}
	return false
}

func GetRequestIP(body []byte, request *http.Request) string {
	ipBytes, _, _, _ := jsonparser.Get(body, "device", "ip")
	if len(ipBytes) > 0 {
		return string(ipBytes)
	}
	return models.GetIP(request)
}

func GetRequestUserAgent(body []byte, request *http.Request) string {
	uaBytes, _, _, _ := jsonparser.Get(body, "device", "ua")
	if len(uaBytes) > 0 {
		return string(uaBytes)
	}
	return request.Header.Get("User-Agent")
}

func getProfileType(partnerConfigMap map[int]map[string]string) int {
	if profileTypeStr, ok := partnerConfigMap[models.VersionLevelConfigID][models.ProfileTypeKey]; ok {
		ProfileType, _ := strconv.Atoi(profileTypeStr)
		return ProfileType
	}
	return 0
}

func getProfileTypePlatform(partnerConfigMap map[int]map[string]string, profileMetaData profilemetadata.ProfileMetaData) int {
	if profileTypePlatformStr, ok := partnerConfigMap[models.VersionLevelConfigID][models.PLATFORM_KEY]; ok {
		if ProfileTypePlatform, ok := profileMetaData.GetProfileTypePlatform(profileTypePlatformStr); ok {
			return ProfileTypePlatform
		}
	}
	return 0
}

func getAppPlatform(partnerConfigMap map[int]map[string]string) int {
	if appPlatformStr, ok := partnerConfigMap[models.VersionLevelConfigID][models.AppPlatformKey]; ok {
		AppPlatform, _ := strconv.Atoi(appPlatformStr)
		return AppPlatform
	}
	return 0
}

func getAppIntegrationPath(partnerConfigMap map[int]map[string]string, profileMetaData profilemetadata.ProfileMetaData) int {
	if appIntegrationPathStr, ok := partnerConfigMap[models.VersionLevelConfigID][models.IntegrationPathKey]; ok {
		if appIntegrationPath, ok := profileMetaData.GetAppIntegrationPath(appIntegrationPathStr); ok {
			return appIntegrationPath
		}
	}
	return -1
}

func getAppSubIntegrationPath(partnerConfigMap map[int]map[string]string, profileMetaData profilemetadata.ProfileMetaData) int {
	if appSubIntegrationPathStr, ok := partnerConfigMap[models.VersionLevelConfigID][models.SubIntegrationPathKey]; ok {
		if appSubIntegrationPath, ok := profileMetaData.GetAppSubIntegrationPath(appSubIntegrationPathStr); ok {
			return appSubIntegrationPath
		}
	}
	if adserverStr, ok := partnerConfigMap[models.VersionLevelConfigID][models.AdserverKey]; ok {
		if adserver, ok := profileMetaData.GetAppSubIntegrationPath(adserverStr); ok {
			return adserver
		}
	}
	return -1
}

func getAccountIdFromRawRequest(hasStoredRequest bool, storedRequest, originalRequest json.RawMessage) (string, bool, bool, []error) {
	request := originalRequest
	if hasStoredRequest {
		request = storedRequest
	}

	accountId, isAppReq, isDOOHReq, err := searchAccountId(request)
	if err != nil {
		return "", isAppReq, isDOOHReq, []error{err}
	}

	// In case the stored request did not have account data we specifically search it in the original request
	if accountId == "" && hasStoredRequest {
		accountId, _, _, err = searchAccountId(originalRequest)
		if err != nil {
			return "", isAppReq, isDOOHReq, []error{err}
		}
	}

	if accountId == "" {
		return metrics.PublisherUnknown, isAppReq, isDOOHReq, nil
	}

	return accountId, isAppReq, isDOOHReq, nil
}

func searchAccountId(request []byte) (string, bool, bool, error) {
	for _, path := range accountIdSearchPath {
		accountId, exists, err := getStringValueFromRequest(request, path.key)
		if err != nil {
			return "", path.isApp, path.isDOOH, err
		}
		if exists {
			return accountId, path.isApp, path.isDOOH, nil
		}
	}
	return "", false, false, nil
}

func getStringValueFromRequest(request []byte, key []string) (string, bool, error) {
	val, dataType, _, err := jsonparser.Get(request, key...)
	if dataType == jsonparser.NotExist {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if dataType != jsonparser.String {
		return "", true, fmt.Errorf("%s must be a string", strings.Join(key, "."))
	}
	return string(val), true, nil
}

func ValidateEIDs(eids []openrtb2.EID) []openrtb2.EID {
	validEIDs := make([]openrtb2.EID, 0, len(eids))
	for _, eid := range eids {
		validUIDs := make([]openrtb2.UID, 0, len(eid.UIDs))
		for _, uid := range eid.UIDs {
			uid.ID = uidRegexp.ReplaceAllString(uid.ID, "")
			if uid.ID != "" {
				validUIDs = append(validUIDs, uid)
			}
		}
		if len(validUIDs) > 0 {
			eid.UIDs = validUIDs
			validEIDs = append(validEIDs, eid)
		}
	}
	return validEIDs
}

func UpdateUserExtWithValidValues(user *openrtb2.User) {
	if user == nil {
		return
	}

	if user.Ext != nil {
		var userExt openrtb_ext.ExtUser
		err := json.Unmarshal(user.Ext, &userExt)
		if err != nil {
			return
		}
		if userExt.SessionDuration < 0 {
			glog.Warningf("Invalid sessionduration value: %v. Only positive values are allowed.", userExt.SessionDuration)
			userExt.SessionDuration = 0
		}

		if userExt.ImpDepth < 0 {
			glog.Warningf("Invalid impdepth value: %v. Only positive values are allowed.", userExt.ImpDepth)
			userExt.ImpDepth = 0
		}
		eids := ValidateEIDs(userExt.Eids)
		userExt.Eids = nil
		if len(eids) > 0 {
			userExt.Eids = eids
		}

		userExtjson, err := json.Marshal(userExt)
		if err == nil {
			user.Ext = userExtjson
		}
	}

	if len(user.EIDs) > 0 {
		eids := ValidateEIDs(user.EIDs)
		user.EIDs = nil
		if len(eids) > 0 {
			user.EIDs = eids
		}
	}
}

func UpdateImpProtocols(impProtocols []adcom1.MediaCreativeSubtype) []adcom1.MediaCreativeSubtype {
	for _, protocol := range protocols {
		if !slices.Contains(impProtocols, protocol) {
			impProtocols = append(impProtocols, protocol)
		}
	}
	return impProtocols
}

func panicHandler(funcName, pubID string) {
	if errInterface := recover(); errInterface != nil {
		ow.GetMetricEngine().RecordOpenWrapServerPanicStats(ow.cfg.Server.HostName, funcName)
		glog.Errorf("stacktrace:[%s], error:[%v], pubid:[%s]", string(debug.Stack()), errInterface, pubID)
		return
	}
}

// getDisplayManagerAndVer returns the display manager and version from the request.app.ext or request.app.prebid.ext source and version
func getDisplayManagerAndVer(app *openrtb2.App) (string, string) {
	if app == nil {
		return "", ""
	}

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

func getAdunitFormat(reward *int8, imp openrtb2.Imp) string {
	if reward != nil && *reward == 1 && imp.Video != nil {
		return models.AdUnitFormatRwddVideo
	}

	if imp.Instl == 1 {
		return models.AdUnitFormatInstl
	}

	if (imp.Banner != nil && imp.Video != nil) ||
		(imp.Banner != nil && reward != nil && *reward == 1) {
		return ""
	}

	if imp.Banner != nil {
		return models.AdUnitFormatBanner
	}
	return ""
}

// getMultiFloors returns all adunitlevel multifloors or to be applied adunitformat multifloors for give imp.
func (m OpenWrap) getMultiFloors(rctx models.RequestCtx, reward *int8, imp openrtb2.Imp) *models.MultiFloors {
	if !sdkutils.IsSdkIntegration(rctx.Endpoint) {
		return nil
	}

	mbmfStatus := models.MBMFNoEntryFound
	//call stat at the end of func with updated mbmfStatus
	defer func() { m.metricEngine.RecordMBMFRequests(rctx.Endpoint, rctx.PubIDStr, mbmfStatus) }()

	// MBMF applies only when the publisher has an explicit MBMF publisher row with is_enabled=1.
	if !m.pubFeatures.IsMBMFPublisherEnabled(rctx.PubID) {
		mbmfStatus = models.MBMFPubDisabled
		return nil
	}

	// check publisher specific country, if not present then check pub_id=0/platform countries
	if !m.pubFeatures.IsMBMFCountryForPublisher(rctx.DeviceCtx.DerivedCountryCode, rctx.PubID) {
		mbmfStatus = models.MBMFCountryDisabled
		return nil
	}

	adunitFormat := getAdunitFormat(reward, imp)
	if adunitFormat == "" {
		mbmfStatus = models.MBMFInvalidAdFormat
		return nil
	}

	// MBMF for this format requires an explicit, active per-format publisher config.
	if !m.pubFeatures.IsMBMFEnabledForAdUnitFormat(rctx.PubID, adunitFormat) {
		mbmfStatus = models.MBMFAdUnitFormatDisabled
		return nil
	}

	adunitLevelMultiFloors := m.pubFeatures.GetProfileAdUnitMultiFloors(rctx.ProfileID)
	if adunitLevelMultiFloors != nil {
		if multifloors, ok := adunitLevelMultiFloors[imp.TagID]; ok && multifloors != nil {
			//if profile adunitlevel floors present and is_active=0, don't apply mbmf
			if !multifloors.IsActive {
				mbmfStatus = models.MBMFAdUnitDisabled
				return nil
			}
			mbmfStatus = models.MBMFSuccess
			return multifloors
		}
	}

	// Banner: only adunit-level floors; no format-level fallback.
	if adunitFormat == models.AdUnitFormatBanner {
		mbmfStatus = models.MBMFAdUnitDisabled
		return nil
	}

	// Rewarded / interstitial: format-level default floors when adunit-level not present.
	multifloors := m.pubFeatures.GetMBMFFloorsForAdUnitFormat(rctx.PubID, adunitFormat)
	if multifloors != nil {
		mbmfStatus = models.MBMFSuccess
		return multifloors
	}
	mbmfStatus = models.MBMFAdUnitFormatNotFound
	return nil
}

// isPrivacyEnforced use should only be limited to VastUnwrap
// isPrivacyEnforced checks if request is privacy enforced if yes then it assumes consent is not given and permits to mask IP.
func isPrivacyEnforced(regs *openrtb2.Regs, device *openrtb2.Device) bool {
	if device != nil && device.Lmt != nil && *device.Lmt == 1 {
		return true
	}

	if regs == nil {
		return false
	}

	if regs.COPPA == 1 || len(regs.USPrivacy) > 0 {
		return true
	}

	// GDPR logic: prefer GPP SID if present
	if len(regs.GPPSID) > 0 {
		if gppPolicy.IsSIDInList(regs.GPPSID, gppConstants.SectionTCFEU2) {
			return true
		}
	} else if regs.GDPR != nil && *regs.GDPR == 1 {
		return true
	}

	var extRegs openrtb_ext.ExtRegs
	if err := json.Unmarshal(regs.Ext, &extRegs); err != nil {
		return false
	}

	if (extRegs.GDPR != nil && *extRegs.GDPR == 1) || len(extRegs.USPrivacy) > 0 {
		return true
	}
	return false
}
