package build

import (
	"encoding/json"

	"github.com/benbjohnson/clock"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/analytics/clients"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/ortb"
	"github.com/prebid/prebid-server/v3/privacy"
)

func New(cfg *config.Analytics) analytics.Runner {
	modules := make(enabledAnalytics)

	// Shared deps for all analytics module builders
	deps := moduledeps.ModuleDeps{
		HTTPClient: clients.GetDefaultHttpInstance(),
		Clock:      clock.New(),
	}

	for vendor, moduleBuilders := range analytics.Builders() {
		for moduleName, buildFn := range moduleBuilders {
			raw := getRawConfigFor(vendor, moduleName, cfg)
			m, err := buildFn(raw, deps)
			if err != nil {
				glog.Errorf("Could not initialize analytics module %s.%s: %v", vendor, moduleName, err)
				continue
			}
			if m != nil {
				// Keep legacy short key used by privacy/activities (e.g. "pubstack", "agma", "filelogger")
				modules[moduleName] = m
			}
		}
	}

	return modules
}

// Collection of all the correctly configured analytics modules - implements the PBSAnalyticsModule interface
type enabledAnalytics map[string]analytics.Module

func (ea enabledAnalytics) LogAuctionObject(ao *analytics.AuctionObject, ac privacy.ActivityControl) {
	for name, module := range ea {
		if isAllowed, cloneBidderReq := evaluateActivities(ao.RequestWrapper, ac, name); isAllowed {
			if cloneBidderReq != nil {
				ao.RequestWrapper = cloneBidderReq
			}
			cloneReq := updateReqWrapperForAnalytics(ao.RequestWrapper, name, cloneBidderReq != nil)
			module.LogAuctionObject(ao)
			if cloneReq != nil {
				ao.RequestWrapper = cloneReq
			}
		}
	}
}

func (ea enabledAnalytics) LogVideoObject(vo *analytics.VideoObject, ac privacy.ActivityControl) {
	for name, module := range ea {
		if isAllowed, cloneBidderReq := evaluateActivities(vo.RequestWrapper, ac, name); isAllowed {
			if cloneBidderReq != nil {
				vo.RequestWrapper = cloneBidderReq
			}
			cloneReq := updateReqWrapperForAnalytics(vo.RequestWrapper, name, cloneBidderReq != nil)
			module.LogVideoObject(vo)
			if cloneReq != nil {
				vo.RequestWrapper = cloneReq
			}
		}

	}
}

func (ea enabledAnalytics) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	for _, module := range ea {
		module.LogCookieSyncObject(cso)
	}
}

func (ea enabledAnalytics) LogSetUIDObject(so *analytics.SetUIDObject) {
	for _, module := range ea {
		module.LogSetUIDObject(so)
	}
}

func (ea enabledAnalytics) LogAmpObject(ao *analytics.AmpObject, ac privacy.ActivityControl) {
	for name, module := range ea {
		if isAllowed, cloneBidderReq := evaluateActivities(ao.RequestWrapper, ac, name); isAllowed {
			if cloneBidderReq != nil {
				ao.RequestWrapper = cloneBidderReq
			}
			cloneReq := updateReqWrapperForAnalytics(ao.RequestWrapper, name, cloneBidderReq != nil)
			module.LogAmpObject(ao)
			if cloneReq != nil {
				ao.RequestWrapper = cloneReq
			}
		}
	}
}

func (ea enabledAnalytics) LogNotificationEventObject(ne *analytics.NotificationEvent, ac privacy.ActivityControl) {
	for name, module := range ea {
		component := privacy.Component{Type: privacy.ComponentTypeAnalytics, Name: name}
		if ac.Allow(privacy.ActivityReportAnalytics, component, privacy.ActivityRequest{}) {
			module.LogNotificationEventObject(ne)
		}
	}
}

// Shutdown - correctly shutdown all analytics modules and wait for them to finish
func (ea enabledAnalytics) Shutdown() {
	for _, module := range ea {
		module.Shutdown()
	}
}

func evaluateActivities(rw *openrtb_ext.RequestWrapper, ac privacy.ActivityControl, componentName string) (bool, *openrtb_ext.RequestWrapper) {
	// returned nil request wrapper means that request wrapper was not modified by activities and doesn't have to be changed in analytics object
	// it is needed in order to use one function for all analytics objects with RequestWrapper
	component := privacy.Component{Type: privacy.ComponentTypeAnalytics, Name: componentName}
	if !ac.Allow(privacy.ActivityReportAnalytics, component, privacy.ActivityRequest{}) {
		return false, nil
	}
	blockUserFPD := !ac.Allow(privacy.ActivityTransmitUserFPD, component, privacy.ActivityRequest{})
	blockPreciseGeo := !ac.Allow(privacy.ActivityTransmitPreciseGeo, component, privacy.ActivityRequest{})

	if !blockUserFPD && !blockPreciseGeo {
		return true, nil
	}

	cloneReq := &openrtb_ext.RequestWrapper{
		BidRequest: ortb.CloneBidRequestPartial(rw.BidRequest),
	}

	if blockUserFPD {
		privacy.ScrubUserFPD(cloneReq)
	}
	if blockPreciseGeo {
		ipConf := privacy.IPConf{IPV6: ac.IPv6Config, IPV4: ac.IPv4Config}
		privacy.ScrubGeoAndDeviceIP(cloneReq, ipConf)
	}

	cloneReq.RebuildRequest()
	return true, cloneReq
}

func updateReqWrapperForAnalytics(rw *openrtb_ext.RequestWrapper, adapterName string, isCloned bool) *openrtb_ext.RequestWrapper {
	if rw == nil {
		return nil
	}
	reqExt, _ := rw.GetRequestExt()
	reqExtPrebid := reqExt.GetPrebid()
	if reqExtPrebid == nil {
		return nil
	}

	var cloneReq *openrtb_ext.RequestWrapper
	if !isCloned {
		cloneReq = &openrtb_ext.RequestWrapper{BidRequest: ortb.CloneBidRequestPartial(rw.BidRequest)}
	} else {
		cloneReq = nil
	}

	if len(reqExtPrebid.Analytics) == 0 {
		return cloneReq
	}

	// Remove the entire analytics object if the adapter module is not present
	if _, ok := reqExtPrebid.Analytics[adapterName]; !ok {
		reqExtPrebid.Analytics = nil
	} else {
		reqExtPrebid.Analytics = updatePrebidAnalyticsMap(reqExtPrebid.Analytics, adapterName)
	}
	reqExt.SetPrebid(reqExtPrebid)
	rw.RebuildRequest()

	if cloneReq != nil {
		cloneReq.RebuildRequest()
	}

	return cloneReq
}

func updatePrebidAnalyticsMap(extPrebidAnalytics map[string]json.RawMessage, adapterName string) map[string]json.RawMessage {
	newMap := make(map[string]json.RawMessage)
	if val, ok := extPrebidAnalytics[adapterName]; ok {
		newMap[adapterName] = val
	}
	return newMap
}

func getRawConfigFor(vendor, module string, cfg *config.Analytics) json.RawMessage {
	if cfg == nil {
		return nil
	}
	switch vendor {
	case "prebid":
		switch module {
		case "filelogger":
			if b, err := json.Marshal(cfg.File); err == nil {
				return b
			}
		case "pubstack":
			if b, err := json.Marshal(cfg.Pubstack); err == nil {
				return b
			}
		case "agma":
			if b, err := json.Marshal(cfg.Agma); err == nil {
				return b
			}
		}
	}
	return nil
}
