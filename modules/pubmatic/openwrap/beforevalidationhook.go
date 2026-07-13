package openwrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	validator "github.com/asaskevich/govalidator"
	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"github.com/prebid/openrtb/v20/adcom1"
	nativeRequests "github.com/prebid/openrtb/v20/native1/request"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/floors"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/adapters"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/adpod"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/adunitconfig"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/bidderparams"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/customdimensions"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/endpoints/legacy/ctv"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	modelsAdunitConfig "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/adunitconfig"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/nbr"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/ortb"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/googlesdk"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/sdkutils"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/utils"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

// logHookBidRequest logs the bidRequest at different stages of the hook execution
func logHookBidRequest(stage string, rCtx models.RequestCtx, bidRequest *openrtb2.BidRequest, nbrCode int) {
	if !glog.V(models.LogLevelDebug) {
		return
	}

	// Lazy marshal: only convert to JSON if we're actually going to log it
	var bidRequestJSON string
	if bidRequestBytes, err := json.Marshal(bidRequest); err == nil {
		bidRequestJSON = string(bidRequestBytes)
	}

	if nbrCode > 0 {
		glog.Infof("[%s] pubid:[%d] profid:[%d] endpoint:[%s] nbr:[%d] bidrequest:[%s]",
			stage, rCtx.PubID, rCtx.ProfileID, rCtx.Endpoint, nbrCode, bidRequestJSON)
	} else {
		glog.Infof("[%s] pubid:[%d] profid:[%d] endpoint:[%s] bidrequest:[%s]",
			stage, rCtx.PubID, rCtx.ProfileID, rCtx.Endpoint, bidRequestJSON)
	}
}

func (m OpenWrap) handleBeforeValidationHook(
	ctx context.Context,
	moduleCtx hookstage.ModuleInvocationContext,
	payload hookstage.BeforeValidationRequestPayload,
) (hookstage.HookResult[hookstage.BeforeValidationRequestPayload], error) {
	result := hookstage.HookResult[hookstage.BeforeValidationRequestPayload]{
		Reject: true,
	}

	if len(moduleCtx.ModuleContext) == 0 {
		result.DebugMessages = append(result.DebugMessages, "error: module-ctx not found in handleBeforeValidationHook()")
		return result, nil
	}
	rCtx, ok := moduleCtx.ModuleContext["rctx"].(models.RequestCtx)
	if !ok {
		result.DebugMessages = append(result.DebugMessages, "error: request-ctx not found in handleBeforeValidationHook()")
		return result, nil
	}

	// Log at the start of the hook
	logHookBidRequest("hook_start", rCtx, payload.BidRequest, 0)

	defer func() {
		moduleCtx.ModuleContext["rctx"] = rCtx

		// Log at the end of the hook with updated bidRequest
		if result.Reject {
			logHookBidRequest("hook_end_rejected", rCtx, payload.BidRequest, result.NbrCode)
			m.metricEngine.RecordBadRequests(rCtx.Endpoint, rCtx.PubIDStr, getPubmaticErrorCode(openrtb3.NoBidReason(result.NbrCode)))
			m.metricEngine.RecordNobidErrPrebidServerRequests(rCtx.PubIDStr, result.NbrCode)
			if rCtx.IsCTVRequest {
				m.metricEngine.RecordCTVInvalidReasonCount(getPubmaticErrorCode(openrtb3.NoBidReason(result.NbrCode)), rCtx.PubIDStr)
			}
		}
	}()

	//Do not execute the module for requests processed in SSHB(8001)
	if rCtx.Sshb == "1" {
		result.Reject = false
		return result, nil
	}

	if rCtx.Endpoint == models.EndpointHybrid {
		//TODO: Add bidder params fix
		result.Reject = false
		return result, nil
	}

	if rCtx.IsCTVRequest {
		m.metricEngine.RecordCTVRequests(rCtx.Endpoint, getPlatformFromRequest(payload.BidRequest))
	}

	// return prebid validation error
	if len(payload.BidRequest.Imp) == 0 || (payload.BidRequest.Site == nil && payload.BidRequest.App == nil) {
		result.Reject = false
		m.metricEngine.RecordBadRequests(rCtx.Endpoint, rCtx.PubIDStr, getPubmaticErrorCode(nbr.InvalidRequestExt))
		m.metricEngine.RecordNobidErrPrebidServerRequests(rCtx.PubIDStr, int(nbr.InvalidRequestExt))
		return result, nil
	}

	rCtx.Source, rCtx.Origin = getSourceAndOrigin(payload.BidRequest)
	rCtx.PageURL = getPageURL(payload.BidRequest)
	rCtx.Platform = getPlatformFromRequest(payload.BidRequest)
	rCtx.DeviceCtx.UA = getUserAgent(payload.BidRequest, rCtx.DeviceCtx.UA)
	rCtx.DeviceCtx.IP = getIP(payload.BidRequest, rCtx.DeviceCtx.IP)
	rCtx.DeviceCtx.Country = getCountry(payload.BidRequest)
	rCtx.DeviceCtx.DerivedCountryCode, _ = m.getCountryCodes(rCtx.DeviceCtx.IP)
	rCtx.DeviceCtx.Platform = getDevicePlatform(rCtx, payload.BidRequest)
	rCtx.IsMaxFloorsEnabled = rCtx.Endpoint == models.EndpointAppLovinMax && m.pubFeatures.IsMaxFloorsEnabled(rCtx.PubID)
	populateDeviceContext(&rCtx.DeviceCtx, payload.BidRequest.Device)

	rCtx.HostName = m.cfg.Server.HostName

	if rCtx.IsCTVRequest {
		m.metricEngine.RecordCTVHTTPMethodRequests(rCtx.Endpoint, rCtx.PubIDStr, rCtx.Method)
	}

	rCtx.IsTBFFeatureEnabled = m.pubFeatures.IsTBFFeatureEnabled(rCtx.PubID, rCtx.ProfileID)

	if rCtx.UidCookie == nil {
		m.metricEngine.RecordUidsCookieNotPresentErrorStats(rCtx.PubIDStr, rCtx.ProfileIDStr)
	}
	m.metricEngine.RecordPublisherProfileRequests(rCtx.PubIDStr, rCtx.ProfileIDStr)

	requestExt, err := models.GetRequestExt(payload.BidRequest.Ext)
	if err != nil {
		result.NbrCode = int(nbr.InvalidRequestExt)
		result.Errors = append(result.Errors, "failed to get request ext: "+err.Error())
		return result, nil
	}
	rCtx.NewReqExt = requestExt
	rCtx.CustomDimensions = customdimensions.GetCustomDimensions(requestExt.Prebid.BidderParams)
	rCtx.ReturnAllBidStatus = requestExt.Prebid.ReturnAllBidStatus
	m.setAnanlyticsFlags(&rCtx)

	// TODO: verify preference of request.test vs queryParam test ++ this check is only for the CTV requests
	if payload.BidRequest.Test != 0 {
		rCtx.IsTestRequest = payload.BidRequest.Test
	}

	partnerConfigMap, err := m.getProfileData(rCtx, *payload.BidRequest)
	if err != nil || len(partnerConfigMap) == 0 {
		// TODO: seperate DB fetch errors as internal errors
		result.NbrCode = int(nbr.InvalidProfileConfiguration)
		rCtx.ImpBidCtx = getDefaultImpBidCtx(*payload.BidRequest) // for wrapper logger sz
		m.metricEngine.RecordPublisherInvalidProfileRequests(rCtx.Endpoint, rCtx.PubIDStr, rCtx.ProfileIDStr)
		return result, errors.New("invalid profile data")
	}

	// Country filter
	if shouldApplyCountryFilter(rCtx.Endpoint) {
		if rCtx.DeviceCtx.DerivedCountryCode != "" {
			mode, countryCodes := getCountryFilterConfig(partnerConfigMap)
			if !isCountryAllowed(rCtx.DeviceCtx.DerivedCountryCode, mode, countryCodes) {
				result.Reject = true
				result.NbrCode = int(nbr.RequestBlockedGeoFiltered)
				result.Errors = append(result.Errors, "Request rejected due to country filter")
				return result, nil
			}
		}
	}

	if rCtx.IsCTVRequest && rCtx.Endpoint == models.EndpointJson {
		if len(rCtx.ResponseFormat) > 0 {
			if rCtx.ResponseFormat != models.ResponseFormatJSON && rCtx.ResponseFormat != models.ResponseFormatRedirect {
				result.NbrCode = int(nbr.InvalidResponseFormat)
				result.Errors = append(result.Errors, "Invalid response format, must be 'json' or 'redirect'")
				return result, nil
			}
		}

		if len(rCtx.RedirectURL) == 0 {
			rCtx.RedirectURL = models.GetVersionLevelPropertyFromPartnerConfig(partnerConfigMap, models.OwRedirectURL)
		}

		if len(rCtx.RedirectURL) > 0 {
			rCtx.RedirectURL = strings.TrimSpace(rCtx.RedirectURL)
			if rCtx.ResponseFormat == models.ResponseFormatRedirect && !isValidURL(rCtx.RedirectURL) {
				result.NbrCode = int(nbr.InvalidRedirectURL)
				result.Errors = append(result.Errors, "Invalid redirect URL")
				return result, nil
			}
		}

		if rCtx.ResponseFormat == models.ResponseFormatRedirect && len(rCtx.RedirectURL) == 0 {
			result.NbrCode = int(nbr.MissingOWRedirectURL)
			result.Errors = append(result.Errors, "owRedirectURL is missing")
			return result, nil
		}
	}

	videoAdDuration := models.GetVersionLevelPropertyFromPartnerConfig(partnerConfigMap, models.VideoAdDurationKey)
	policy := models.GetVersionLevelPropertyFromPartnerConfig(partnerConfigMap, models.VideoAdDurationMatchingKey)
	if len(videoAdDuration) > 0 {
		rCtx.AdpodProfileConfig = &models.AdpodProfileConfig{
			AdserverCreativeDurations:              utils.GetIntArrayFromString(videoAdDuration, models.ArraySeparator),
			AdserverCreativeDurationMatchingPolicy: policy,
		}
	}

	rCtx.PartnerConfigMap = partnerConfigMap // keep a copy at module level as well
	if ver, err := strconv.Atoi(models.GetVersionLevelPropertyFromPartnerConfig(partnerConfigMap, models.DisplayVersionID)); err == nil {
		rCtx.DisplayVersionID = ver
	}
	platform := rCtx.GetVersionLevelKey(models.PLATFORM_KEY)
	if platform == "" {
		result.NbrCode = int(nbr.InvalidPlatform)
		rCtx.ImpBidCtx = getDefaultImpBidCtx(*payload.BidRequest) // for wrapper logger sz
		m.metricEngine.RecordPublisherInvalidProfileRequests(rCtx.Endpoint, rCtx.PubIDStr, rCtx.ProfileIDStr)
		return result, errors.New("failed to get platform data")
	}
	rCtx.Platform = platform
	rCtx.DeviceCtx.Platform = getDevicePlatform(rCtx, payload.BidRequest)
	rCtx.SendAllBids = isSendAllBids(rCtx)

	m.metricEngine.RecordPublisherRequests(rCtx.Endpoint, rCtx.PubIDStr, rCtx.Platform)

	if newPartnerConfigMap, ok := ABTestProcessing(rCtx); ok {
		rCtx.ABTestConfigApplied = 1
		rCtx.PartnerConfigMap = newPartnerConfigMap
		result.Warnings = append(result.Warnings, "update the rCtx.PartnerConfigMap with ABTest data")
	}

	//set the profile MetaData for logging and tracking
	rCtx.ProfileType = getProfileType(partnerConfigMap)
	rCtx.ProfileTypePlatform = getProfileTypePlatform(partnerConfigMap, m.profileMetaData)
	rCtx.AppPlatform = getAppPlatform(partnerConfigMap)
	rCtx.AppIntegrationPath = ptrutil.ToPtr(getAppIntegrationPath(partnerConfigMap, m.profileMetaData))
	rCtx.AppSubIntegrationPath = ptrutil.ToPtr(getAppSubIntegrationPath(partnerConfigMap, m.profileMetaData))

	// To check if VAST unwrap needs to be enabled for given request
	if isVastUnwrapEnabled(rCtx.PartnerConfigMap, m.cfg.Features.VASTUnwrapPercent) {
		//rCtx.ABTestConfigApplied = 1 // Re-use AB Test flag for VAST unwrap feature
		rCtx.VastUnWrap.Enabled = true
		rCtx.VastUnWrap.IsPrivacyEnforced = isPrivacyEnforced(payload.BidRequest.Regs, payload.BidRequest.Device)
	}

	//TMax should be updated after ABTest processing
	rCtx.TMax = m.setTimeout(rCtx, payload.BidRequest)

	allPartnersThrottledFlag := false
	rCtx.AdapterThrottleMap, allPartnersThrottledFlag = m.applyPartnerThrottling(rCtx)

	if allPartnersThrottledFlag {
		result.NbrCode = int(nbr.RequestBlockedGeoFiltered)
		result.Errors = append(result.Errors, "All adapters Blocked due to Geo Filtering")
		rCtx.ImpBidCtx = getDefaultImpBidCtx(*payload.BidRequest) // for wrapper logger sz
		glog.V(models.LogLevelDebug).Info("All adapters Blocked due to Geo Filtering")
		return result, nil
	}

	var allPartnersFilteredFlag bool

	rCtx.AdapterThrottleMap, allPartnersThrottledFlag = GetAdapterThrottleMap(rCtx.PartnerConfigMap, rCtx.AdapterThrottleMap)

	if allPartnersThrottledFlag {
		result.NbrCode = int(nbr.AllPartnerThrottled)
		result.Errors = append(result.Errors, "All adapters throttled")
		rCtx.ImpBidCtx = getDefaultImpBidCtx(*payload.BidRequest) // for wrapper logger sz
		return result, nil
	}

	rCtx.AdapterFilteredMap, allPartnersFilteredFlag = m.getFilteredBidders(rCtx, payload.BidRequest)

	result.SeatNonBid = getSeatNonBid(rCtx.AdapterFilteredMap, payload)

	if allPartnersFilteredFlag {
		result.NbrCode = int(nbr.AllPartnersFiltered)
		result.Errors = append(result.Errors, "All partners filtered")
		rCtx.ImpBidCtx = getDefaultImpBidCtx(*payload.BidRequest) // for wrapper logger sz
		return result, err
	}

	priceGranularity, err := computePriceGranularity(rCtx)
	if err != nil {
		result.NbrCode = int(nbr.InvalidPriceGranularityConfig)
		result.Errors = append(result.Errors, "failed to price granularity details: "+err.Error())
		rCtx.ImpBidCtx = getDefaultImpBidCtx(*payload.BidRequest) // for wrapper logger sz
		return result, nil
	}

	rCtx.PriceGranularity = &priceGranularity
	rCtx.AdUnitConfig = m.cache.GetAdunitConfigFromCache(payload.BidRequest, rCtx.PubID, rCtx.ProfileID, rCtx.DisplayID)

	requestExt.Prebid.Debug = rCtx.Debug
	requestExt.Prebid.DebugOverride = rCtx.WakandaDebug.IsEnable()
	requestExt.Prebid.SupportDeals = rCtx.SupportDeals && rCtx.IsCTVRequest // TODO: verify usecase of Prefered deals vs Support details
	requestExt.Prebid.ExtOWRequestPrebid.TrackerDisabled = rCtx.TrackerDisabled
	requestExt.Prebid.AlternateBidderCodes, rCtx.MarketPlaceBidders = getMarketplaceBidders(requestExt.Prebid.AlternateBidderCodes, partnerConfigMap)
	requestExt.Prebid.Targeting = &openrtb_ext.ExtRequestTargeting{
		PriceGranularity:  &priceGranularity,
		IncludeBidderKeys: ptrutil.ToPtr(true),
		IncludeWinners:    ptrutil.ToPtr(true),
	}
	// TODO: Check if we can directly accept keyVal in prebid ext
	if requestExt.Wrapper != nil && requestExt.Wrapper.KeyValues != nil {
		requestExt.Prebid.KeyVal = requestExt.Wrapper.KeyValues
	}
	setIncludeBrandCategory(requestExt.Wrapper, &requestExt.Prebid, partnerConfigMap, rCtx.IsCTVRequest)

	disabledSlots := 0
	serviceSideBidderPresent := false
	requestExt.Prebid.BidAdjustmentFactors = map[string]float64{}
	// Get currency rates conversions and store in rctx for tracker/logger calculation
	conversions := currency.GetAuctionCurrencyRates(m.rateConvertor, requestExt.Prebid.CurrencyConversions)
	rCtx.CurrencyConversion = func(from, to string, value float64) (float64, error) {
		rate, err := conversions.GetRate(from, to)
		if err == nil {
			return value * rate, nil
		}
		return 0, err
	}

	if rCtx.IsCTVRequest {
		err := ctv.ValidateVideoImpressions(payload.BidRequest)
		if err != nil {
			result.NbrCode = int(nbr.InvalidVideoRequest)
			result.Errors = append(result.Errors, err.Error())
			rCtx.ImpBidCtx = getDefaultImpBidCtx(*payload.BidRequest) // for wrapper logger sz
			return result, nil
		}
	}
	displaymanager, displaymanagerVer := getDisplayManagerAndVer(payload.BidRequest.App)

	aliasgvlids := make(map[string]uint16)
	rCtx.MultiFloors = make(map[string]*models.MultiFloors)
	for i := 0; i < len(payload.BidRequest.Imp); i++ {
		slotType := "banner"
		imp := payload.BidRequest.Imp[i]

		impExt := &models.ImpExtension{}
		if len(imp.Ext) != 0 {
			err := json.Unmarshal(imp.Ext, impExt)
			if err != nil {
				result.NbrCode = int(openrtb3.NoBidInvalidRequest)
				err = errors.New("failed to parse imp.ext: " + imp.ID)
				result.Errors = append(result.Errors, err.Error())
				rCtx.ImpBidCtx = map[string]models.ImpCtx{} // do not create "s" object in owlogger
				return result, err
			}
		}
		if rCtx.Endpoint == models.EndpointWebS2S {
			imp.TagID = getTagID(imp, impExt)
		}
		if imp.TagID == "" {
			result.NbrCode = int(nbr.InvalidImpressionTagID)
			err = errors.New("tagid missing for imp: " + imp.ID)
			result.Errors = append(result.Errors, err.Error())
			rCtx.ImpBidCtx = map[string]models.ImpCtx{} // do not create "s" object in owlogger
			return result, err
		}

		div := ""
		if impExt.Wrapper != nil {
			div = impExt.Wrapper.Div
		}

		// reuse the existing impExt instead of allocating a new one
		var reward *int8
		if imp.Rwdd == 1 {
			reward = openrtb2.Int8Ptr(1)
		}
		if reward == nil && impExt.Reward != nil {
			reward = impExt.Reward
			impExt.Prebid.IsRewardedInventory = reward
		}
		// if imp.ext.data.pbadslot is absent then set it to tagId
		if len(impExt.Data.PbAdslot) == 0 {
			impExt.Data.PbAdslot = imp.TagID
		}

		// Add size 300x600 for interstitial banner
		if (sdkutils.IsSdkIntegration(rCtx.Endpoint) || rCtx.Endpoint == models.EndpointV25) && imp.Instl == 1 {
			sdkutils.AddSize300x600ForInterstitialBanner(&imp)
		}

		var videoAdUnitCtx, bannerAdUnitCtx, nativeAdUnitCtx models.AdUnitCtx
		if rCtx.AdUnitConfig != nil {
			if (rCtx.Platform == models.PLATFORM_APP || rCtx.Platform == models.PLATFORM_VIDEO || rCtx.Platform == models.PLATFORM_DISPLAY) && imp.Video != nil {
				if payload.BidRequest.App != nil && payload.BidRequest.App.Content != nil {
					m.metricEngine.RecordReqImpsWithContentCount(rCtx.PubIDStr, models.ContentTypeApp)
				}
				if payload.BidRequest.Site != nil && payload.BidRequest.Site.Content != nil {
					m.metricEngine.RecordReqImpsWithContentCount(rCtx.PubIDStr, models.ContentTypeSite)
				}
			}

			var connectionType *adcom1.ConnectionType
			if payload.BidRequest.Device != nil && payload.BidRequest.Device.ConnectionType != nil {
				connectionType = payload.BidRequest.Device.ConnectionType
			}

			videoAdUnitCtx = adunitconfig.UpdateVideoObjectWithAdunitConfig(rCtx, imp, div, connectionType)
			if rCtx.Endpoint == models.EndpointAMP && m.pubFeatures.IsAmpMultiformatEnabled(rCtx.PubID) && isVideoEnabledForAMP(videoAdUnitCtx.AppliedSlotAdUnitConfig) {
				//Iniitalized local imp.Video object to update macros and get mappings in case of AMP request
				rCtx.AmpVideoEnabled = true
				imp.Video = &openrtb2.Video{}
			}
			//banner can not be disabled for AMP requests through adunit config
			if rCtx.Endpoint != models.EndpointAMP {
				bannerAdUnitCtx = adunitconfig.UpdateBannerObjectWithAdunitConfig(rCtx, imp, div)
			}
			nativeAdUnitCtx = adunitconfig.UpdateNativeObjectWithAdunitConfig(rCtx, imp, div)
		}

		if imp.Video != nil {
			slotType = "video"

			//add stats for video instl impressions
			if imp.Instl == 1 {
				m.metricEngine.RecordVideoInstlImpsStats(rCtx.PubIDStr, rCtx.ProfileIDStr)
			}
			if len(requestExt.Prebid.Macros) == 0 {
				// provide custom macros for video event trackers
				requestExt.Prebid.Macros = getVASTEventMacros(rCtx)
			}

			if rCtx.IsCTVRequest && imp.Video.Ext != nil {
				if _, _, _, err := jsonparser.Get(imp.Video.Ext, "adpod"); err == nil {
					m.metricEngine.RecordCTVReqCountWithAdPod(rCtx.PubIDStr, rCtx.ProfileIDStr)
				}
			}
		}

		incomingSlots := getIncomingSlots(imp, videoAdUnitCtx)
		slotName := getSlotName(imp.TagID, impExt)
		adUnitName := getAdunitName(imp.TagID, impExt)

		// ignore adunit config status for native as it is not supported for native
		if !isSlotEnabled(imp, videoAdUnitCtx, bannerAdUnitCtx, nativeAdUnitCtx) {
			disabledSlots++

			rCtx.ImpBidCtx[imp.ID] = models.ImpCtx{ // for wrapper logger sz
				IncomingSlots:     incomingSlots,
				AdUnitName:        adUnitName,
				SlotName:          slotName,
				IsRewardInventory: reward,
			}
			continue
		}

		var adpodConfig *models.AdPod
		if rCtx.IsCTVRequest {
			adpodConfig, err = adpod.GetV25AdpodConfigs(imp.Video, requestExt.AdPod, videoAdUnitCtx.AppliedSlotAdUnitConfig, partnerConfigMap, rCtx.PubIDStr, m.metricEngine)
			if err != nil {
				result.NbrCode = int(nbr.InvalidAdpodConfig)
				result.Errors = append(result.Errors, "failed to get adpod configurations for "+imp.ID+" reason: "+err.Error())
				rCtx.ImpBidCtx = getDefaultImpBidCtx(*payload.BidRequest)
				return result, nil
			}

			//Adding default durations for CTV Test requests
			if rCtx.IsTestRequest > 0 && adpodConfig != nil && rCtx.AdpodProfileConfig == nil {
				rCtx.AdpodProfileConfig = &models.AdpodProfileConfig{
					AdserverCreativeDurations:              []int{5, 10},
					AdserverCreativeDurationMatchingPolicy: openrtb_ext.OWRoundupVideoAdDurationMatching,
				}
			}

			if err := adpod.ValidateV25Configs(rCtx, adpodConfig); err != nil {
				result.NbrCode = int(nbr.InvalidAdpodConfig)
				result.Errors = append(result.Errors, "invalid adpod configurations for "+imp.ID+" reason: "+err.Error())
				rCtx.ImpBidCtx = getDefaultImpBidCtx(*payload.BidRequest)
				return result, nil
			}

			podConfigs, err := adpod.GetAdpodConfigs(rCtx, m.cache, videoAdUnitCtx.AppliedSlotAdUnitConfig)
			if err != nil {
				result.NbrCode = int(nbr.InvalidAdpodConfig)
				result.Errors = append(result.Errors, "failed to get adpod configurations for "+imp.ID+" reason: "+err.Error())
				rCtx.ImpBidCtx = getDefaultImpBidCtx(*payload.BidRequest)
				return result, nil
			}

			err = adpod.ValidateAdpodConfigs(podConfigs)
			if err != nil {
				result.NbrCode = int(nbr.InvalidAdpodConfig)
				result.Errors = append(result.Errors, "invalid adpod configurations for "+imp.ID+" reason: "+err.Error())
				rCtx.ImpBidCtx = getDefaultImpBidCtx(*payload.BidRequest)
				return result, nil
			}

			if len(podConfigs) > 0 {
				rCtx.ImpAdPodConfig[imp.ID] = podConfigs
			}
		}

		rCtx.MultiFloors[imp.ID] = m.getMultiFloors(rCtx, reward, imp)
		bidderMeta := make(map[string]models.PartnerData)
		nonMapped := make(map[string]struct{})
		for _, partnerConfig := range rCtx.PartnerConfigMap {
			if partnerConfig[models.SERVER_SIDE_FLAG] != "1" {
				continue
			}

			partneridstr, ok := partnerConfig[models.PARTNER_ID]
			if !ok {
				continue
			}
			partnerID, err := strconv.Atoi(partneridstr)
			if err != nil || partnerID == models.VersionLevelConfigID {
				continue
			}

			// bidderCode is in context with pubmatic. Ex. it could be appnexus-1, appnexus-2, etc.
			bidderCode := partnerConfig[models.BidderCode]
			// prebidBidderCode is equivalent of PBS-Core's bidderCode
			prebidBidderCode := partnerConfig[models.PREBID_PARTNER_NAME]
			//
			rCtx.PrebidBidderCode[bidderCode] = prebidBidderCode

			if _, ok := rCtx.AdapterFilteredMap[bidderCode]; ok {
				result.Warnings = append(result.Warnings, "Dropping adapter due to bidder filtering: "+bidderCode)
				continue
			}

			if _, ok := rCtx.AdapterThrottleMap[bidderCode]; ok {
				result.Warnings = append(result.Warnings, "Dropping throttled adapter from auction: "+bidderCode)
				continue
			}

			var (
				isRegex             bool
				slot, kgpv          string
				bidderParams        json.RawMessage
				matchedSlotKeysVAST []string
			)
			switch prebidBidderCode {
			case string(openrtb_ext.BidderPubmatic), models.BidderPubMaticSecondaryAlias:
				slot, kgpv, isRegex, bidderParams, err = bidderparams.PreparePubMaticParamsV25(rCtx, m.cache, *payload.BidRequest, imp, *impExt, partnerID)
			case models.BidderVASTBidder:
				slot, bidderParams, matchedSlotKeysVAST, err = bidderparams.PrepareVASTBidderParams(rCtx, m.cache, *payload.BidRequest, imp, *impExt, partnerID, adpodConfig)
			default:
				slot, kgpv, isRegex, bidderParams, err = bidderparams.PrepareAdapterParamsV25(rCtx, m.cache, *payload.BidRequest, imp, *impExt, partnerID)
			}

			if err != nil || len(bidderParams) == 0 {
				result.Errors = append(result.Errors, fmt.Sprintf("no bidder params found for imp:%s partner: %s", imp.ID, prebidBidderCode))
				nonMapped[bidderCode] = struct{}{}
				m.metricEngine.RecordPartnerConfigErrors(rCtx.PubIDStr, rCtx.ProfileIDStr, bidderCode, models.PartnerErrSlotNotMapped)

				if prebidBidderCode != string(openrtb_ext.BidderPubmatic) && prebidBidderCode != string(models.BidderPubMaticSecondaryAlias) {
					continue
				}
			}

			m.metricEngine.RecordPlatformPublisherPartnerReqStats(rCtx.Platform, rCtx.PubIDStr, bidderCode)

			if requestExt.Prebid.SupportDeals && impExt.Bidder != nil {
				var bidderParamsMap map[string]interface{}
				err := json.Unmarshal(bidderParams, &bidderParamsMap)
				if err == nil {
					if bidderExt, ok := impExt.Bidder[bidderCode]; ok && bidderExt != nil && bidderExt.DealTier != nil {
						bidderParamsMap[models.DEAL_TIER_KEY] = bidderExt.DealTier
					}
					newBidderParams, err := json.Marshal(bidderParamsMap)
					if err == nil {
						bidderParams = newBidderParams
					}
				}
			}

			bidderMeta[bidderCode] = models.PartnerData{
				PartnerID:        partnerID,
				PrebidBidderCode: prebidBidderCode,
				MatchedSlot:      slot, // KGPSV
				Params:           bidderParams,
				KGP:              rCtx.PartnerConfigMap[partnerID][models.KEY_GEN_PATTERN], // acutual slot
				KGPV:             kgpv,                                                     // regex pattern, use this field for pubmatic default unmapped slot as well using isRegex
				IsRegex:          isRegex,                                                  // regex pattern
			}

			if len(matchedSlotKeysVAST) > 0 {
				meta := bidderMeta[bidderCode]
				meta.VASTTagFlags = make(map[string]bool)
				bidderMeta[bidderCode] = meta
			}

			isAlias := false
			if alias, ok := partnerConfig[models.IsAlias]; ok && alias == "1" {
				if prebidPartnerName, ok := partnerConfig[models.PREBID_PARTNER_NAME]; ok {
					rCtx.Aliases[bidderCode] = adapters.ResolveOWBidder(prebidPartnerName)
					isAlias = true
				}
			}
			if alias, ok := IsAlias(bidderCode); ok {
				rCtx.Aliases[bidderCode] = alias
				isAlias = true
			}

			if isAlias || partnerConfig[models.PREBID_PARTNER_NAME] == models.BidderVASTBidder {
				updateAliasGVLIds(aliasgvlids, bidderCode, partnerConfig)
			}

			revShare := models.GetRevenueShare(rCtx.PartnerConfigMap[partnerID])
			requestExt.Prebid.BidAdjustmentFactors[bidderCode] = models.GetBidAdjustmentValue(revShare)
			serviceSideBidderPresent = true
		} // for(rctx.PartnerConfigMap

		// update the imp.ext with bidder params for this
		if impExt.Prebid.Bidder == nil {
			impExt.Prebid.Bidder = make(map[string]json.RawMessage)
		}
		for bidder, meta := range bidderMeta {
			impExt.Prebid.Bidder[bidder] = meta.Params
		}
		adserverURL := ""
		if impExt.Wrapper != nil {
			adserverURL = impExt.Wrapper.AdServerURL
		}

		if rCtx.Endpoint == models.EndpointAppLovinMax || rCtx.Endpoint == models.EndpointUnityLevelPlay {
			if len(impExt.GpId) == 0 {
				impExt.GpId = imp.TagID
			}
		}

		if sdkutils.IsSdkIntegration(rCtx.Endpoint) {
			appStoreUrl, isValidAppStoreUrl := getProfileAppStoreUrl(rCtx)
			if !isValidAppStoreUrl && payload.BidRequest.App != nil && payload.BidRequest.App.StoreURL != "" {
				appStoreUrl = payload.BidRequest.App.StoreURL
			}
			rCtx.AppStoreUrl = appStoreUrl
			rCtx.PageURL = appStoreUrl
			if appStoreUrl != "" {
				m.updateSkadnSourceapp(rCtx, payload.BidRequest, impExt)
			}
		}

		isCTAOverlayRequest := impExt.OWSDK != nil && impExt.OWSDK["ctaoverlay"] == float64(1)
		impExt.Wrapper = nil
		impExt.Reward = nil
		impExt.Bidder = nil
		impExt.OWSDK = nil
		newImpExt, err := json.Marshal(impExt)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to update bidder params for impression %s", imp.ID))
		}

		if imp.DisplayManager != "" {
			displaymanager = imp.DisplayManager
		}
		if imp.DisplayManagerVer != "" {
			displaymanagerVer = imp.DisplayManagerVer
		}
		// cache the details for further processing
		if _, ok := rCtx.ImpBidCtx[imp.ID]; !ok {
			rCtx.ImpBidCtx[imp.ID] = models.ImpCtx{
				ImpID:             imp.ID,
				TagID:             imp.TagID,
				Div:               div,
				IsRewardInventory: reward,
				BidFloor:          imp.BidFloor,
				BidFloorCur:       imp.BidFloorCur,
				Type:              slotType,
				IsBanner:          imp.Banner != nil,
				Banner:            ortb.DeepCopyImpBanner(imp.Banner),
				Video:             imp.Video,
				Native:            imp.Native,
				IncomingSlots:     incomingSlots,
				Bidders:           make(map[string]models.PartnerData),
				BidCtx:            make(map[string]models.BidCtx),
				NewExt:            json.RawMessage(newImpExt),
				AdpodConfig:       adpodConfig,
				SlotName:          slotName,
				AdUnitName:        adUnitName,
				AdserverURL:       adserverURL,
				DisplayManager:    displaymanager,
				DisplayManagerVer: displaymanagerVer,
			}
		}

		impCtx := rCtx.ImpBidCtx[imp.ID]
		impCtx.Instl = imp.Instl
		impCtx.IsAppOpenAd = impExt.IsAppOpenAd
		impCtx.IsCTAOverlayRequest = isCTAOverlayRequest
		impCtx.Bidders = bidderMeta
		impCtx.NonMapped = nonMapped
		impCtx.VideoAdUnitCtx = videoAdUnitCtx
		impCtx.BannerAdUnitCtx = bannerAdUnitCtx
		impCtx.NativeAdUnitCtx = nativeAdUnitCtx
		rCtx.ImpBidCtx[imp.ID] = impCtx
	} // for(imp

	if disabledSlots == len(payload.BidRequest.Imp) {
		result.NbrCode = int(nbr.AllSlotsDisabled)
		if err != nil {
			err = errors.New("all slots disabled: " + err.Error())
		} else {
			err = errors.New("all slots disabled")
		}
		result.Errors = append(result.Errors, err.Error())
		return result, nil
	}

	if !serviceSideBidderPresent {
		result.NbrCode = int(nbr.ServerSidePartnerNotConfigured)
		if err != nil {
			err = errors.New("server side partner not found: " + err.Error())
		} else {
			err = errors.New("server side partner not found")
		}
		result.Errors = append(result.Errors, err.Error())
		return result, nil
	}

	if cto := setContentTransparencyObject(rCtx, requestExt); cto != nil {
		requestExt.Prebid.Transparency = cto
	}

	adunitconfig.UpdateFloorsExtObjectFromAdUnitConfig(rCtx, requestExt)
	setFloorsExt(requestExt, &rCtx, m.pubFeatures.IsDynamicFloorEnabledPublisher(rCtx.PubID))

	if len(rCtx.Aliases) != 0 && requestExt.Prebid.Aliases == nil {
		requestExt.Prebid.Aliases = make(map[string]string)
	}
	for k, v := range rCtx.Aliases {
		requestExt.Prebid.Aliases[k] = v
	}

	requestExt.Prebid.AliasGVLIDs = aliasgvlids
	if _, ok := rCtx.AdapterThrottleMap[string(openrtb_ext.BidderPubmatic)]; !ok {
		requestExt.Prebid.BidderParams, _ = updateRequestExtBidderParamsPubmatic(requestExt.Prebid.BidderParams, rCtx.Cookies, rCtx.LoggerImpressionID, string(openrtb_ext.BidderPubmatic), rCtx.SendBurl, rCtx.AppSubIntegrationPath)
	}

	for bidderCode, coreBidder := range rCtx.Aliases {
		if coreBidder == string(openrtb_ext.BidderPubmatic) {
			if _, ok := rCtx.AdapterThrottleMap[bidderCode]; !ok {
				requestExt.Prebid.BidderParams, _ = updateRequestExtBidderParamsPubmatic(requestExt.Prebid.BidderParams, rCtx.Cookies, rCtx.LoggerImpressionID, bidderCode, rCtx.SendBurl, rCtx.AppSubIntegrationPath)
			}
		}
	}
	rCtx.GoogleSDK.SDKRenderedAdID = googlesdk.SetSDKRenderedAdID(payload.BidRequest.App, rCtx.Endpoint)

	// similar to impExt, reuse the existing requestExt to avoid additional memory requests
	requestExt.Wrapper = nil
	requestExt.Bidder = nil

	// if rCtx.Debug {
	// 	newImp, _ := json.Marshal(rCtx.ImpBidCtx)
	// 	result.DebugMessages = append(result.DebugMessages, "new imp: "+string(newImp))
	// 	newReqExt, _ := json.Marshal(rCtx.NewReqExt)
	// 	result.DebugMessages = append(result.DebugMessages, "new request.ext: "+string(newReqExt))
	// }

	result.ChangeSet.AddMutation(func(ep hookstage.BeforeValidationRequestPayload) (hookstage.BeforeValidationRequestPayload, error) {
		rctx := moduleCtx.ModuleContext["rctx"].(models.RequestCtx)
		defer func() {
			moduleCtx.ModuleContext["rctx"] = rctx
			logHookBidRequest("hook_end_success", rCtx, ep.BidRequest, 0)

			// Always record preprocessing time stats
			timeDiff := time.Since(time.Unix(rCtx.StartTime, 0)).Milliseconds()
			m.metricEngine.RecordPreProcessingTimeStats(rCtx.PubIDStr, int(timeDiff))

			// Debug logging only
			if glog.V(models.LogLevelDebug) {
				processingTime := time.Duration(timeDiff) * time.Millisecond
				timeoutDuration := time.Duration(rCtx.TMax) * time.Millisecond
				remainingTime := timeoutDuration - processingTime
				glog.Infof("[%s] Total processing time taken before auction: %v", rCtx.LoggerImpressionID, processingTime)
				glog.Infof("[%s] Max Timeout set: %v, Prebid Delta set: %v", rCtx.LoggerImpressionID, timeoutDuration, m.cfg.Timeout.PrebidDelta)
				glog.Infof("[%s] Remaining time for the auction: %v", rCtx.LoggerImpressionID, remainingTime)
			}
		}()

		var err error
		if rctx.IsCTVRequest && ep.BidRequest.Source != nil && ep.BidRequest.Source.SChain != nil {
			err = ctv.IsValidSchain(ep.BidRequest.Source.SChain)
			if err != nil {
				schainBytes, _ := json.Marshal(ep.BidRequest.Source.SChain)
				glog.Errorf(ctv.ErrSchainValidationFailed, models.SChainKey, err.Error(), rctx.PubIDStr, rctx.ProfileIDStr, string(schainBytes))
				ep.BidRequest.Source.SChain = nil
			}
		}
		ep.BidRequest, err = m.applyProfileChanges(rctx, ep.BidRequest)
		if err != nil {
			result.Errors = append(result.Errors, "failed to apply profile changes: "+err.Error())
		}

		if rctx.Endpoint == models.EndpointAppLovinMax && ep.BidRequest.Source != nil {
			m.updateAppLovinMaxRequestSchain(&rctx, ep.BidRequest)
		}

		if rctx.IsCTVRequest {
			err = ctv.FilterNonVideoImpressions(ep.BidRequest)
			if err != nil {
				result.Errors = append(result.Errors, err.Error())
			}

			ep.BidRequest = adpod.ApplyAdpodConfigs(rctx, ep.BidRequest)
		}
		return ep, err
	}, hookstage.MutationUpdate, "request-body-with-profile-data")

	result.Reject = false
	return result, nil
}

// applyProfileChanges copies and updates BidRequest with required values from http header and partnetConfigMap
func (m *OpenWrap) applyProfileChanges(rctx models.RequestCtx, bidRequest *openrtb2.BidRequest) (*openrtb2.BidRequest, error) {
	if rctx.IsTestRequest > 0 {
		bidRequest.Test = 1
	}

	if sdkutils.IsSdkIntegration(rctx.Endpoint) && rctx.AppStoreUrl != "" {
		bidRequest.App.StoreURL = rctx.AppStoreUrl
	}

	// Remove app.ext.token
	if rctx.Endpoint == models.EndpointUnityLevelPlay {
		bidRequest.App.Ext = jsonparser.Delete(bidRequest.App.Ext, "token")
	}

	googleSSUFeatureEnabled := models.GetVersionLevelPropertyFromPartnerConfig(rctx.PartnerConfigMap, models.GoogleSSUFeatureEnabledKey) == models.Enabled
	if googleSSUFeatureEnabled {
		if rctx.NewReqExt == nil {
			rctx.NewReqExt = &models.RequestExt{}
		}
		rctx.NewReqExt.Prebid.GoogleSSUFeatureEnabled = googleSSUFeatureEnabled
	}
	if cur, ok := rctx.PartnerConfigMap[models.VersionLevelConfigID][models.AdServerCurrency]; ok {
		bidRequest.Cur = append(bidRequest.Cur, cur)
	}
	if bidRequest.TMax == 0 {
		bidRequest.TMax = rctx.TMax
	}

	if bidRequest.Source == nil {
		bidRequest.Source = &openrtb2.Source{}
	}
	bidRequest.Source.TID = bidRequest.ID

	for i := 0; i < len(bidRequest.Imp); i++ {
		if rctx.Endpoint != models.EndpointAMP {
			m.applyBannerAdUnitConfig(rctx, &bidRequest.Imp[i])
		}
		m.applyVideoAdUnitConfig(rctx, &bidRequest.Imp[i])
		m.applyNativeAdUnitConfig(rctx, &bidRequest.Imp[i])
		m.applyImpChanges(rctx, &bidRequest.Imp[i])
	}

	setSChainInRequest(rctx.NewReqExt, bidRequest.Source, rctx.PartnerConfigMap)

	adunitconfig.ReplaceAppObjectFromAdUnitConfig(rctx, bidRequest.App)
	adunitconfig.ReplaceDeviceTypeFromAdUnitConfig(rctx, &bidRequest.Device)
	bidRequest.Device.IP = rctx.DeviceCtx.IP
	bidRequest.Device.Language = getValidLanguage(bidRequest.Device.Language)
	amendDeviceObject(bidRequest.Device, &rctx.DeviceCtx)

	if bidRequest.User == nil {
		bidRequest.User = &openrtb2.User{}
	}
	if bidRequest.User.CustomData == "" && rctx.KADUSERCookie != nil {
		bidRequest.User.CustomData = rctx.KADUSERCookie.Value
	}
	UpdateUserExtWithValidValues(bidRequest.User)

	for i := 0; i < len(bidRequest.WLang); i++ {
		bidRequest.WLang[i] = getValidLanguage(bidRequest.WLang[i])
	}

	if bidRequest.Site != nil && bidRequest.Site.Content != nil {
		bidRequest.Site.Content.Language = getValidLanguage(bidRequest.Site.Content.Language)
	} else if bidRequest.App != nil && bidRequest.App.Content != nil {
		bidRequest.App.Content.Language = getValidLanguage(bidRequest.App.Content.Language)
	}

	var err error
	var requestExtjson json.RawMessage
	if rctx.NewReqExt != nil {
		requestExtjson, err = json.Marshal(rctx.NewReqExt)
		bidRequest.Ext = requestExtjson
	}
	return bidRequest, err
}

func (m *OpenWrap) applyVideoAdUnitConfig(rCtx models.RequestCtx, imp *openrtb2.Imp) {
	//For AMP request, if AmpVideoEnabled is true then crate a empty video object and update with adunitConfigs
	if rCtx.AmpVideoEnabled {
		imp.Video = &openrtb2.Video{}
	}

	if imp.Video == nil {
		return
	}

	adUnitCfg := rCtx.ImpBidCtx[imp.ID].VideoAdUnitCtx.AppliedSlotAdUnitConfig
	if adUnitCfg == nil {
		return
	}

	impBidCtx := rCtx.ImpBidCtx[imp.ID]
	imp.BidFloor, imp.BidFloorCur = getImpBidFloorParams(rCtx, adUnitCfg, imp, m.rateConvertor.Rates())
	impBidCtx.BidFloor = imp.BidFloor
	impBidCtx.BidFloorCur = imp.BidFloorCur

	rCtx.ImpBidCtx[imp.ID] = impBidCtx

	if adUnitCfg.Exp != nil {
		imp.Exp = int64(*adUnitCfg.Exp)
	}

	if adUnitCfg.Video == nil {
		return
	}

	//check if video is disabled, if yes then remove video from imp object
	if adUnitCfg.Video.Enabled != nil && !*adUnitCfg.Video.Enabled {
		imp.Video = nil
		impBidCtx.Video = nil
		rCtx.ImpBidCtx[imp.ID] = impBidCtx
		return
	}

	//For AMP request if AmpVideoEnabled is true then, update the imp.video object with adunitConfig and if adunitConfig is not present then update with default values
	if rCtx.AmpVideoEnabled {
		if adUnitCfg.Video.Config != nil {
			updateImpVideoWithVideoConfig(imp, adUnitCfg.Video.Config)
		}
		updateAmpImpVideoWithDefault(imp)
		return
	}

	if adUnitCfg.Video.Config != nil {
		updateImpVideoWithVideoConfig(imp, adUnitCfg.Video.Config)
	}
}

func (m *OpenWrap) applyImpChanges(rCtx models.RequestCtx, imp *openrtb2.Imp) {
	if imp.BidFloor == 0 {
		imp.BidFloorCur = ""
	} else if imp.BidFloorCur == "" {
		imp.BidFloorCur = models.USD
	}

	if imp.Video != nil {
		m.applyImpVideoChanges(rCtx, imp.Video)
	}

	//update secure for applovin
	if rCtx.Endpoint == models.EndpointAppLovinMax {
		imp.Secure = openrtb2.Int8Ptr(1)
	}

	//update impression extensions
	imp.Ext = rCtx.ImpBidCtx[imp.ID].NewExt
}

func (m *OpenWrap) applyImpVideoChanges(rCtx models.RequestCtx, video *openrtb2.Video) {
	//update protocols
	if rCtx.NewReqExt != nil && rCtx.NewReqExt.Prebid.GoogleSSUFeatureEnabled {
		video.Protocols = UpdateImpProtocols(video.Protocols)
	}

	//update video.plcmt from video.placements
	if video.Placement > 0 && video.Plcmt == 0 {
		//TODO: move to ConvertUpTo26 once upgraded to prebid3.x version
		switch video.Placement {
		case adcom1.VideoPlacementInStream:
			video.Plcmt = adcom1.VideoPlcmtInstream
		case adcom1.VideoPlacementInBanner:
			video.Plcmt = adcom1.VideoPlcmtNoContent
		case adcom1.VideoPlacementAlwaysVisible:
			video.Plcmt = adcom1.VideoPlcmtInterstitial
		}
	}
}

func getImpBidFloorParams(rCtx models.RequestCtx, adUnitCfg *modelsAdunitConfig.AdConfig, imp *openrtb2.Imp, conversions currency.Conversions) (float64, string) {
	bidfloor := imp.BidFloor
	bidfloorcur := imp.BidFloorCur

	if rCtx.IsMaxFloorsEnabled && adUnitCfg.BidFloor != nil {
		bidfloor, bidfloorcur, _ = floors.GetMaxFloorValue(bidfloor, bidfloorcur, *adUnitCfg.BidFloor, *adUnitCfg.BidFloorCur, conversions)
	} else {
		if bidfloor == 0 && adUnitCfg.BidFloor != nil {
			//use adunitconfig bidfloor and bidfloorcur
			bidfloor = *adUnitCfg.BidFloor
			bidfloorcur = ""

			if adUnitCfg.BidFloorCur != nil {
				bidfloorcur = *adUnitCfg.BidFloorCur
			}
		}
	}

	if bidfloor == 0 {
		//no bidfloor value
		return 0, ""
	}
	if bidfloorcur == "" {
		bidfloorcur = models.USD
	}
	return bidfloor, bidfloorcur
}

func (m *OpenWrap) applyBannerAdUnitConfig(rCtx models.RequestCtx, imp *openrtb2.Imp) {
	if imp.Banner == nil {
		return
	}

	adUnitCfg := rCtx.ImpBidCtx[imp.ID].BannerAdUnitCtx.AppliedSlotAdUnitConfig
	if adUnitCfg == nil {
		return
	}

	impBidCtx := rCtx.ImpBidCtx[imp.ID]
	imp.BidFloor, imp.BidFloorCur = getImpBidFloorParams(rCtx, adUnitCfg, imp, m.rateConvertor.Rates())
	impBidCtx.BidFloor = imp.BidFloor
	impBidCtx.BidFloorCur = imp.BidFloorCur
	rCtx.ImpBidCtx[imp.ID] = impBidCtx

	if adUnitCfg.Exp != nil {
		imp.Exp = int64(*adUnitCfg.Exp)
	}

	if adUnitCfg.Banner == nil {
		return
	}

	if adUnitCfg.Banner.Enabled != nil && !*adUnitCfg.Banner.Enabled {
		imp.Banner = nil
		return
	}
}

// isVastUnwrapEnabled return whether to enable vastunwrap or not
func isVastUnwrapEnabled(partnerConfigMap map[int]map[string]string, vastUnwrapTraffic int) bool {
	trafficPercentage := vastUnwrapTraffic
	unwrapEnabled := models.GetVersionLevelPropertyFromPartnerConfig(partnerConfigMap, models.VastUnwrapperEnableKey) == models.Enabled
	if unwrapEnabled {
		if value := models.GetVersionLevelPropertyFromPartnerConfig(partnerConfigMap, models.VastUnwrapTrafficPercentKey); len(value) > 0 {
			if trafficPercentDB, err := strconv.Atoi(value); err == nil {
				trafficPercentage = trafficPercentDB
			}
		}
	}
	return unwrapEnabled && GetRandomNumberIn1To100() <= trafficPercentage
}

/*
getSlotName will return slot name according to below priority
 1. imp.ext.gpid
 2. imp.tagid
 3. imp.ext.data.pbadslot
 4. imp.ext.prebid.storedrequest.id
*/
func getSlotName(tagId string, impExt *models.ImpExtension) string {
	if impExt == nil {
		return tagId
	}

	if len(impExt.GpId) > 0 {
		return impExt.GpId
	}

	if len(tagId) > 0 {
		return tagId
	}

	if len(impExt.Data.PbAdslot) > 0 {
		return impExt.Data.PbAdslot
	}

	var storeReqId string
	if impExt.Prebid.StoredRequest != nil {
		storeReqId = impExt.Prebid.StoredRequest.ID
	}

	return storeReqId
}

/*
getAdunitName will return adunit name according to below priority
 1. imp.ext.data.adserver.adslot if imp.ext.data.adserver.name == "gam"
 2. imp.ext.data.pbadslot
 3. imp.tagid
*/
func getAdunitName(tagId string, impExt *models.ImpExtension) string {
	if impExt == nil {
		return tagId
	}
	if impExt.Data.AdServer != nil && impExt.Data.AdServer.Name == models.GamAdServer && impExt.Data.AdServer.AdSlot != "" {
		return impExt.Data.AdServer.AdSlot
	}
	if len(impExt.Data.PbAdslot) > 0 {
		return impExt.Data.PbAdslot
	}
	return tagId
}

func getDomainFromUrl(pageUrl string) string {
	u, err := url.Parse(pageUrl)
	if err != nil {
		return ""
	}

	return u.Host
}

// always perfer rCtx.LoggerImpressionID received in request. Create a new once if it is not availble.
// func getLoggerID(reqExt models.ExtRequestWrapper) string {
// 	if reqExt.Wrapper.LoggerImpressionID != "" {
// 		return reqExt.Wrapper.LoggerImpressionID
// 	}
// 	return uuid.NewV4().String()
// }

// NYC: make this generic. Do we need this?. PBS now has auto_gen_source_tid generator. We can make it to wiid for pubmatic adapter in pubmatic.go
// Rebuilds ext.prebid.bidderparams.<bidderCode> for OW: replaces the whole bidder object (only wiid, optional wrapper, Cookie, sendburl).
// Incoming keys for that bidder are not preserved. Profile and version are not carried on request ext here; they live on
// imp.ext.prebid.bidder.pubmatic.wrapper (WrapExt). So bidderparams.wrapper is only created to hold sdksubintegration for the adapter, not a merge of a full client wrapper.
func updateRequestExtBidderParamsPubmatic(bidderParams json.RawMessage, cookie []string, loggerID, bidderCode string, sendBurl bool, sdkSubIntegration *int) (json.RawMessage, error) {
	bidderParamsMap := make(map[string]map[string]interface{})
	_ = json.Unmarshal(bidderParams, &bidderParamsMap) // ignore error, incoming might be nil for now but we still have data to put

	bidderParamsMap[bidderCode] = map[string]interface{}{
		models.WrapperLoggerImpID: loggerID,
	}
	if sdkSubIntegration != nil && *sdkSubIntegration >= 0 {
		// Minimal wrapper on bidderparams only (see function comment); not a merge with client request ext.
		bidderParamsMap[bidderCode]["wrapper"] = map[string]interface{}{
			"sdksubintegration": *sdkSubIntegration,
		}
	}

	if len(cookie) != 0 {
		bidderParamsMap[bidderCode][models.COOKIE] = cookie
	}

	if sendBurl {
		bidderParamsMap[bidderCode][models.SendBurl] = true
	}

	return json.Marshal(bidderParamsMap)
}

func getPageURL(bidRequest *openrtb2.BidRequest) string {
	if bidRequest.App != nil && bidRequest.App.StoreURL != "" {
		return bidRequest.App.StoreURL
	} else if bidRequest.Site != nil && bidRequest.Site.Page != "" {
		return bidRequest.Site.Page
	}
	return ""
}

// getVASTEventMacros populates macros with PubMatic specific macros
// These marcros is used in replacing with actual values of Macros in case of Video Event tracke URLs
// If this function fails to determine value of any macro then it continues with next macro setup
// returns true when at least one macro is added to map
func getVASTEventMacros(rctx models.RequestCtx) map[string]string {
	macros := map[string]string{
		string(models.MacroProfileID):           fmt.Sprintf("%d", rctx.ProfileID),
		string(models.MacroProfileVersionID):    fmt.Sprintf("%d", rctx.DisplayVersionID),
		string(models.MacroUnixTimeStamp):       fmt.Sprintf("%d", rctx.StartTime),
		string(models.MacroPlatform):            fmt.Sprintf("%d", rctx.DeviceCtx.Platform),
		string(models.MacroWrapperImpressionID): rctx.LoggerImpressionID,
	}

	if rctx.SSAI != "" {
		macros[string(models.MacroSSAI)] = rctx.SSAI
	}

	return macros
}

func updateAliasGVLIds(aliasgvlids map[string]uint16, bidderCode string, partnerConfig map[string]string) {
	if vendorID, ok := partnerConfig[models.VENDORID]; ok && vendorID != "" {
		vid, err := strconv.ParseUint(vendorID, 10, 64)
		if err != nil {
			return
		}

		if vid == 0 {
			return
		}
		aliasgvlids[bidderCode] = uint16(vid)
	}
}

// setTimeout - This utility returns timeout applicable for a profile
func (m OpenWrap) setTimeout(rCtx models.RequestCtx, req *openrtb2.BidRequest) int64 {
	var auctionTimeout int64

	// BidRequest.TMax has highest priority
	if req.TMax != 0 {
		auctionTimeout = req.TMax
	} else {
		//check for ssTimeout in the partner config
		ssTimeout := models.GetVersionLevelPropertyFromPartnerConfig(rCtx.PartnerConfigMap, models.SSTimeoutKey)
		if ssTimeout != "" {
			ssTimeoutDB, err := strconv.Atoi(ssTimeout)
			if err == nil {
				auctionTimeout = int64(ssTimeoutDB)
			}
		}
	}

	// found tmax value in request or db
	if auctionTimeout != 0 {
		if auctionTimeout < m.cfg.Timeout.MinTimeout {
			return m.cfg.Timeout.MinTimeout
		} else if auctionTimeout > m.cfg.Timeout.MaxTimeout {
			return m.cfg.Timeout.MaxTimeout
		}
		return auctionTimeout
	}

	//Below piece of code is applicable for older profiles where ssTimeout is not set
	//Here we will check the partner timeout and select max timeout considering timeout range
	auctionTimeout = m.cfg.Timeout.MinTimeout
	for _, partnerConfig := range rCtx.PartnerConfigMap {
		partnerTO, _ := strconv.Atoi(partnerConfig[models.TIMEOUT])
		if int64(partnerTO) > m.cfg.Timeout.MaxTimeout {
			auctionTimeout = m.cfg.Timeout.MaxTimeout
			break
		}
		if int64(partnerTO) >= m.cfg.Timeout.MinTimeout && auctionTimeout < int64(partnerTO) {
			auctionTimeout = int64(partnerTO)

		}
	}
	return auctionTimeout
}

// isSendAllBids returns true in below cases:
// if ssauction flag is set 0 in the request
// if ssauction flag is not set and platform is dislay, then by default send all bids
// if ssauction flag is not set and platform is in-app, then check if profile setting sendAllBids is set to 1
func isSendAllBids(rctx models.RequestCtx) bool {
	//for webs2s endpoint SendAllBids is always true
	if rctx.Endpoint == models.EndpointWebS2S {
		return true
	}
	//if ssauction is set to 0 in the request
	if rctx.SSAuction == 0 {
		return true
	} else if rctx.SSAuction == -1 && rctx.Platform == models.PLATFORM_APP {
		// if platform is in-app, then check if profile setting sendAllBids is set to 1
		if models.GetVersionLevelPropertyFromPartnerConfig(rctx.PartnerConfigMap, models.SendAllBidsKey) == "1" {
			return true
		}
	}
	return false
}

func getValidLanguage(language string) string {
	if len(language) > 2 {
		lang := language[0:2]
		if models.ValidCode(lang) {
			return lang
		}
	}
	return language
}

func isSlotEnabled(imp openrtb2.Imp, videoAdUnitCtx, bannerAdUnitCtx, nativeAdUnitCtx models.AdUnitCtx) bool {
	videoEnabled := true
	if imp.Video == nil || (videoAdUnitCtx.AppliedSlotAdUnitConfig != nil && videoAdUnitCtx.AppliedSlotAdUnitConfig.Video != nil &&
		videoAdUnitCtx.AppliedSlotAdUnitConfig.Video.Enabled != nil && !*videoAdUnitCtx.AppliedSlotAdUnitConfig.Video.Enabled) {
		videoEnabled = false
	}

	bannerEnabled := true
	if imp.Banner == nil || (bannerAdUnitCtx.AppliedSlotAdUnitConfig != nil && bannerAdUnitCtx.AppliedSlotAdUnitConfig.Banner != nil &&
		bannerAdUnitCtx.AppliedSlotAdUnitConfig.Banner.Enabled != nil && !*bannerAdUnitCtx.AppliedSlotAdUnitConfig.Banner.Enabled) {
		bannerEnabled = false
	}

	nativeEnabled := true
	if imp.Native == nil || (nativeAdUnitCtx.AppliedSlotAdUnitConfig != nil && nativeAdUnitCtx.AppliedSlotAdUnitConfig.Native != nil &&
		nativeAdUnitCtx.AppliedSlotAdUnitConfig.Native.Enabled != nil && !*nativeAdUnitCtx.AppliedSlotAdUnitConfig.Native.Enabled) {
		nativeEnabled = false
	}

	return videoEnabled || bannerEnabled || nativeEnabled
}

func getPubID(bidRequest openrtb2.BidRequest) (pubID int, err error) {

	if bidRequest.Site != nil && bidRequest.Site.Publisher != nil && bidRequest.Site.Publisher.ID != "" {
		pubID, err = strconv.Atoi(bidRequest.Site.Publisher.ID)
	} else if bidRequest.App != nil && bidRequest.App.Publisher != nil && bidRequest.App.Publisher.ID != "" {
		pubID, err = strconv.Atoi(bidRequest.App.Publisher.ID)
	}
	return pubID, err
}

func getTagID(imp openrtb2.Imp, impExt *models.ImpExtension) string {
	//priority for tagId is imp.ext.gpid > imp.TagID > imp.ext.data.pbadslot
	if impExt.GpId != "" {
		if idx := strings.Index(impExt.GpId, "#"); idx != -1 {
			return impExt.GpId[:idx]
		}
		return impExt.GpId
	} else if imp.TagID != "" {
		return imp.TagID
	}
	return impExt.Data.PbAdslot
}

func (m OpenWrap) setAnanlyticsFlags(rCtx *models.RequestCtx) {
	rCtx.LoggerDisabled, rCtx.TrackerDisabled = m.pubFeatures.IsAnalyticsTrackingThrottled(rCtx.PubID, rCtx.ProfileID)

	if rCtx.LoggerDisabled {
		rCtx.MetricsEngine.RecordAnalyticsTrackingThrottled(strconv.Itoa(rCtx.PubID), strconv.Itoa(rCtx.ProfileID), models.AnanlyticsThrottlingLoggerType)
	}

	if rCtx.TrackerDisabled {
		rCtx.MetricsEngine.RecordAnalyticsTrackingThrottled(strconv.Itoa(rCtx.PubID), strconv.Itoa(rCtx.ProfileID), models.AnanlyticsThrottlingTrackerType)
	}
}

func updateImpVideoWithVideoConfig(imp *openrtb2.Imp, configObjInVideoConfig *modelsAdunitConfig.VideoConfig) {
	if len(imp.Video.MIMEs) == 0 {
		imp.Video.MIMEs = configObjInVideoConfig.MIMEs
	}

	if imp.Video.MinDuration == 0 {
		imp.Video.MinDuration = configObjInVideoConfig.MinDuration
	}

	if imp.Video.MaxDuration == 0 {
		imp.Video.MaxDuration = configObjInVideoConfig.MaxDuration
	}

	if imp.Video.Skip == nil {
		imp.Video.Skip = configObjInVideoConfig.Skip
	}

	if imp.Video.SkipMin == 0 {
		imp.Video.SkipMin = configObjInVideoConfig.SkipMin
	}

	if imp.Video.SkipAfter == 0 {
		imp.Video.SkipAfter = configObjInVideoConfig.SkipAfter
	}

	if len(imp.Video.BAttr) == 0 {
		imp.Video.BAttr = configObjInVideoConfig.BAttr
	}

	if imp.Video.MinBitRate == 0 {
		imp.Video.MinBitRate = configObjInVideoConfig.MinBitRate
	}

	if imp.Video.MaxBitRate == 0 {
		imp.Video.MaxBitRate = configObjInVideoConfig.MaxBitRate
	}

	if imp.Video.MaxExtended == 0 {
		imp.Video.MaxExtended = configObjInVideoConfig.MaxExtended
	}

	if imp.Video.StartDelay == nil {
		imp.Video.StartDelay = configObjInVideoConfig.StartDelay
	}

	if imp.Video.Placement == 0 {
		imp.Video.Placement = configObjInVideoConfig.Placement
	}

	if imp.Video.Plcmt == 0 {
		imp.Video.Plcmt = configObjInVideoConfig.Plcmt
	}

	if imp.Video.Linearity == 0 {
		imp.Video.Linearity = configObjInVideoConfig.Linearity
	}

	if imp.Video.Protocol == 0 {
		imp.Video.Protocol = configObjInVideoConfig.Protocol
	}

	if len(imp.Video.Protocols) == 0 {
		imp.Video.Protocols = configObjInVideoConfig.Protocols
	}

	if imp.Video.W == nil {
		imp.Video.W = configObjInVideoConfig.W
	}

	if imp.Video.H == nil {
		imp.Video.H = configObjInVideoConfig.H
	}

	if imp.Video.Sequence == 0 {
		imp.Video.Sequence = configObjInVideoConfig.Sequence
	}

	if imp.Video.BoxingAllowed == nil {
		imp.Video.BoxingAllowed = configObjInVideoConfig.BoxingAllowed
	}

	if len(imp.Video.PlaybackMethod) == 0 {
		imp.Video.PlaybackMethod = configObjInVideoConfig.PlaybackMethod
	}

	if imp.Video.PlaybackEnd == 0 {
		imp.Video.PlaybackEnd = configObjInVideoConfig.PlaybackEnd
	}

	if imp.Video.Delivery == nil {
		imp.Video.Delivery = configObjInVideoConfig.Delivery
	}

	if imp.Video.Pos == nil {
		imp.Video.Pos = configObjInVideoConfig.Pos
	}

	if len(imp.Video.API) == 0 {
		imp.Video.API = configObjInVideoConfig.API
	}

	if len(imp.Video.CompanionType) == 0 {
		imp.Video.CompanionType = configObjInVideoConfig.CompanionType
	}

	if imp.Video.CompanionAd == nil {
		imp.Video.CompanionAd = configObjInVideoConfig.CompanionAd
	}
}

func updateAmpImpVideoWithDefault(imp *openrtb2.Imp) {
	if imp.Video.W == nil {
		imp.Video.W = getW(imp)
	}
	if imp.Video.H == nil {
		imp.Video.H = getH(imp)
	}
	if imp.Video.MIMEs == nil {
		imp.Video.MIMEs = []string{"video/mp4"}
	}
	if imp.Video.MinDuration == 0 {
		imp.Video.MinDuration = 0
	}
	if imp.Video.MaxDuration == 0 {
		imp.Video.MaxDuration = 30
	}
	if imp.Video.StartDelay == nil {
		imp.Video.StartDelay = adcom1.StartPreRoll.Ptr()
	}
	if imp.Video.Protocols == nil {
		imp.Video.Protocols = []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST10, adcom1.CreativeVAST20, adcom1.CreativeVAST30, adcom1.CreativeVAST10Wrapper, adcom1.CreativeVAST20Wrapper, adcom1.CreativeVAST30Wrapper, adcom1.CreativeVAST40, adcom1.CreativeVAST40Wrapper, adcom1.CreativeVAST41, adcom1.CreativeVAST41Wrapper, adcom1.CreativeVAST42, adcom1.CreativeVAST42Wrapper}
	}
	if imp.Video.Placement == 0 {
		imp.Video.Placement = adcom1.VideoPlacementInBanner
	}
	if imp.Video.Plcmt == 0 {
		imp.Video.Plcmt = adcom1.VideoPlcmtNoContent
	}
	if imp.Video.Linearity == 0 {
		imp.Video.Linearity = adcom1.LinearityLinear
	}
	if imp.Video.Skip == nil {
		imp.Video.Skip = ptrutil.ToPtr[int8](0)
	}
	if imp.Video.PlaybackMethod == nil {
		imp.Video.PlaybackMethod = []adcom1.PlaybackMethod{adcom1.PlaybackPageLoadSoundOff}
	}
	if imp.Video.PlaybackEnd == 0 {
		imp.Video.PlaybackEnd = adcom1.PlaybackCompletion
	}
	if imp.Video.Delivery == nil {
		imp.Video.Delivery = []adcom1.DeliveryMethod{adcom1.DeliveryProgressive, adcom1.DeliveryDownload}
	}
}

func getW(imp *openrtb2.Imp) *int64 {
	if imp.Banner != nil {
		if imp.Banner.W != nil {
			return imp.Banner.W
		}
		for _, format := range imp.Banner.Format {
			if format.W != 0 {
				return &format.W
			}
		}
	}
	return nil
}

func getH(imp *openrtb2.Imp) *int64 {
	if imp.Banner != nil {
		if imp.Banner.H != nil {
			return imp.Banner.H
		}
		for _, format := range imp.Banner.Format {
			if format.H != 0 {
				return &format.H
			}
		}
	}
	return nil
}

func isValidURL(urlVal string) bool {
	if !(strings.HasPrefix(urlVal, "http://") || strings.HasPrefix(urlVal, "https://")) {
		return false
	}
	return validator.IsRequestURL(urlVal) && validator.IsURL(urlVal)
}

func getProfileAppStoreUrl(rctx models.RequestCtx) (string, bool) {
	isValidAppStoreUrl := false
	appStoreUrl := rctx.PartnerConfigMap[models.VersionLevelConfigID][models.AppStoreUrl]
	if appStoreUrl == "" {
		glog.Errorf("[PubID]: %d [ProfileID]: %d [Error]: app store url not present in DB", rctx.PubID, rctx.ProfileID)
		return appStoreUrl, isValidAppStoreUrl
	}
	appStoreUrl = strings.TrimSpace(appStoreUrl)
	if !isValidURL(appStoreUrl) {
		glog.Errorf("[PubID]: %d [ProfileID]: %d [AppStoreUrl]: %s [Error]: Invalid app store url", rctx.PubID, rctx.ProfileID, appStoreUrl)
		return appStoreUrl, isValidAppStoreUrl
	}
	isValidAppStoreUrl = true
	return appStoreUrl, isValidAppStoreUrl
}

func (m *OpenWrap) updateSkadnSourceapp(rctx models.RequestCtx, bidRequest *openrtb2.BidRequest, impExt *models.ImpExtension) {
	if bidRequest.Device == nil || strings.ToLower(bidRequest.Device.OS) != "ios" {
		return
	}

	if impExt == nil || impExt.SKAdnetwork == nil {
		glog.Errorf("[PubID]: %d [ProfileID]: %d [Error]: skadn is missing in imp.ext", rctx.PubID, rctx.ProfileID)
		return
	}

	itunesID := extractItunesIdFromAppStoreUrl(rctx.AppStoreUrl)
	if itunesID == "" {
		m.metricEngine.RecordFailedParsingItuneID(rctx.PubIDStr, rctx.ProfileIDStr)
		glog.Errorf("[PubID]: %d [ProfileID]: %d [AppStoreUrl]: %s [Error]: itunes id is missing in app store url", rctx.PubID, rctx.ProfileID, rctx.AppStoreUrl)
		return
	}

	if updatedSKAdnetwork, err := jsonparser.Set(impExt.SKAdnetwork, []byte(strconv.Quote(itunesID)), "sourceapp"); err != nil {
		glog.Errorf("[PubID]: %d [ProfileID]: %d [AppStoreUrl]: %s [Error]: %s", rctx.PubID, rctx.ProfileID, rctx.AppStoreUrl, err.Error())
	} else {
		impExt.SKAdnetwork = updatedSKAdnetwork
	}
}

func extractItunesIdFromAppStoreUrl(url string) string {
	url = strings.TrimSuffix(url, "/")
	itunesID := ""
	for i := len(url) - 1; i >= 0; i-- {
		char := rune(url[i])
		if unicode.IsDigit(char) {
			itunesID = string(char) + itunesID
		} else {
			break
		}
	}
	return itunesID
}

func (m *OpenWrap) applyNativeAdUnitConfig(rCtx models.RequestCtx, imp *openrtb2.Imp) {
	if imp.Native == nil {
		return
	}

	impCtx, ok := rCtx.ImpBidCtx[imp.ID]
	if !ok {
		return
	}
	adUnitCfg := impCtx.NativeAdUnitCtx.AppliedSlotAdUnitConfig
	if adUnitCfg == nil {
		return
	}

	impBidCtx := rCtx.ImpBidCtx[imp.ID]
	imp.BidFloor, imp.BidFloorCur = getImpBidFloorParams(rCtx, adUnitCfg, imp, m.rateConvertor.Rates())
	impBidCtx.BidFloor = imp.BidFloor
	impBidCtx.BidFloorCur = imp.BidFloorCur
	rCtx.ImpBidCtx[imp.ID] = impBidCtx

	if adUnitCfg.Exp != nil {
		imp.Exp = int64(*adUnitCfg.Exp)
	}

	if adUnitCfg.Native == nil {
		return
	}

	if adUnitCfg.Native.Enabled != nil && !*adUnitCfg.Native.Enabled {
		imp.Native = nil
		return
	}
	applyNativeVideoAssetRulesFromAdUnitConfig(adUnitCfg.Native.Config, imp.Native, rCtx.PubID, rCtx.ProfileID)
}

// applyNativeVideoAssetRulesFromAdUnitConfig applies native video asset rules from adunit config, if configured
func applyNativeVideoAssetRulesFromAdUnitConfig(nativeCfg *modelsAdunitConfig.NativeConfig, impNative *openrtb2.Native, PubID int, ProfileID int) {
	if impNative.Request == "" || nativeCfg == nil || nativeCfg.Video.Enabled == nil {
		return
	}

	var nReq nativeRequests.Request
	if err := json.Unmarshal([]byte(impNative.Request), &nReq); err != nil {
		glog.Errorf("[native_request_json_unmarshal_failed][PubID]: %d [ProfileID]: %d [Error]: %s", PubID, ProfileID, err.Error())
		return
	}

	assets := nReq.Assets
	changed := false
	if *nativeCfg.Video.Enabled {
		videoCfg := nativeCfg.Video.Config
		for i := range assets {
			if assets[i].Video == nil {
				continue
			}
			if videoCfg.MinDuration != nil {
				assets[i].Video.MinDuration = *videoCfg.MinDuration
				changed = true
			}
			if videoCfg.MaxDuration != nil {
				assets[i].Video.MaxDuration = *videoCfg.MaxDuration
				changed = true
			}
		}
	} else {
		writeIdx := 0
		for i := range assets {
			if assets[i].Video != nil {
				changed = true
				continue
			}
			assets[writeIdx] = assets[i]
			writeIdx++
		}
		if changed {
			nReq.Assets = assets[:writeIdx]
		}
	}
	if !changed {
		return
	}
	nReqBytes, err := json.Marshal(&nReq)
	if err != nil {
		glog.Errorf("[native_request_json_marshal_failed][PubID]: %d [ProfileID]: %d [Error]: %s", PubID, ProfileID, err.Error())
	} else {
		impNative.Request = string(nReqBytes)
	}
}
