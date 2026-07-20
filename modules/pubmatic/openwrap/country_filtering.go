package openwrap

import (
	"strings"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/sdkutils"
)

func getCountryFilterConfig(partnerConfigMap map[int]map[string]string) (mode string, countryCodes string) {
	mode = models.GetVersionLevelPropertyFromPartnerConfig(partnerConfigMap, models.CountryFilterModeKey)
	if mode == "" {
		return "", ""
	}

	countryCodes = models.GetVersionLevelPropertyFromPartnerConfig(partnerConfigMap, models.CountryCodesKey)
	return mode, countryCodes

}

func isCountryAllowed(country string, mode string, countryCodes string) bool {
	if mode == "" || countryCodes == "" {
		return true
	}

	found := strings.Contains(countryCodes, country)

	// For allowlist (mode "1"), return true if country is found
	// For blocklist (mode "0"), return true if country is not found
	return (mode == "1" && found) || (mode == "0" && !found)
}

func shouldApplyCountryFilter(endpoint string) bool {
	return sdkutils.IsSdkBiddingEndpoint(endpoint)
}

func (m *OpenWrap) applyPartnerThrottling(rCtx models.RequestCtx) (map[string]struct{}, bool) {

	throttleMap, err := m.cache.GetThrottlePartnersWithCriteria(rCtx.DeviceCtx.DerivedCountryCode)
	if err != nil {
		glog.Errorf("Error getting throttled partners for country %s: %v", rCtx.DeviceCtx.DerivedCountryCode, err)
		return nil, false
	}
	if len(throttleMap) == 0 {
		return nil, false
	}

	adapterThrottleMap := make(map[string]struct{})
	allPartnersThrottledFlag := true
	for _, cfg := range rCtx.PartnerConfigMap {
		if cfg[models.SERVER_SIDE_FLAG] != "1" {
			continue
		}
		bidderCode, ok := cfg[models.BidderCode]
		if !ok || bidderCode == "" {
			continue
		}

		if _, isThrottled := throttleMap[bidderCode]; isThrottled {
			// 90% of throttled traffic is still allowed through for testing or monitoring purposes
			if GetRandomNumberIn1To100() <= m.cfg.Features.AllowPartnerLevelThrottlingPercentage {
				glog.V(models.LogLevelDebug).Infof("Allowing %f %% fallback traffic for throttled bidder: %s", m.cfg.Features.AllowPartnerLevelThrottlingPercentage, bidderCode)
				allPartnersThrottledFlag = false
				continue
			}
			adapterThrottleMap[bidderCode] = struct{}{}
			m.metricEngine.RecordPartnerThrottledRequests(rCtx.PubIDStr, bidderCode, models.PartnerLevelThrottlingFeatureID)
			m.metricEngine.RecordCountryLevelPartnerThrottledRequests(rCtx.Endpoint, bidderCode, rCtx.DeviceCtx.DerivedCountryCode)
		} else if allPartnersThrottledFlag {
			allPartnersThrottledFlag = false
		}
	}
	return adapterThrottleMap, allPartnersThrottledFlag
}
