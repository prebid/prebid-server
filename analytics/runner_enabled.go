package analytics

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/ortb"
	"github.com/prebid/prebid-server/v3/privacy"
)

// EnabledAnalytics: kolekcja poprawnie skonfigurowanych modułów analytics.
type EnabledAnalytics map[string]Module

func (ea EnabledAnalytics) LogAuctionObject(ao *AuctionObject, ac privacy.ActivityControl) {
	for name, module := range ea {
		if isAllowed, cloneBidderReq := EvaluateActivities(ao.RequestWrapper, ac, name); isAllowed {
			if cloneBidderReq != nil {
				ao.RequestWrapper = cloneBidderReq
			}
			cloneReq := UpdateReqWrapperForAnalytics(ao.RequestWrapper, name, cloneBidderReq != nil)
			module.LogAuctionObject(ao)
			if cloneReq != nil {
				ao.RequestWrapper = cloneReq
			}
		}
	}
}

func (ea EnabledAnalytics) LogVideoObject(vo *VideoObject, ac privacy.ActivityControl) {
	for name, module := range ea {
		if isAllowed, cloneBidderReq := EvaluateActivities(vo.RequestWrapper, ac, name); isAllowed {
			if cloneBidderReq != nil {
				vo.RequestWrapper = cloneBidderReq
			}
			cloneReq := UpdateReqWrapperForAnalytics(vo.RequestWrapper, name, cloneBidderReq != nil)
			module.LogVideoObject(vo)
			if cloneReq != nil {
				vo.RequestWrapper = cloneReq
			}
		}
	}
}

func (ea EnabledAnalytics) LogCookieSyncObject(cso *CookieSyncObject) {
	for _, module := range ea {
		module.LogCookieSyncObject(cso)
	}
}

func (ea EnabledAnalytics) LogSetUIDObject(so *SetUIDObject) {
	for _, module := range ea {
		module.LogSetUIDObject(so)
	}
}

func (ea EnabledAnalytics) LogAmpObject(ao *AmpObject, ac privacy.ActivityControl) {
	for name, module := range ea {
		if isAllowed, cloneBidderReq := EvaluateActivities(ao.RequestWrapper, ac, name); isAllowed {
			if cloneBidderReq != nil {
				ao.RequestWrapper = cloneBidderReq
			}
			cloneReq := UpdateReqWrapperForAnalytics(ao.RequestWrapper, name, cloneBidderReq != nil)
			module.LogAmpObject(ao)
			if cloneReq != nil {
				ao.RequestWrapper = cloneReq
			}
		}
	}
}

func (ea EnabledAnalytics) LogNotificationEventObject(ne *NotificationEvent, ac privacy.ActivityControl) {
	for name, module := range ea {
		component := privacy.Component{Type: privacy.ComponentTypeAnalytics, Name: name}
		if ac.Allow(privacy.ActivityReportAnalytics, component, privacy.ActivityRequest{}) {
			module.LogNotificationEventObject(ne)
		}
	}
}

func (ea EnabledAnalytics) Shutdown() {
	for _, module := range ea {
		module.Shutdown()
	}
}

// Exportowane helpery (używane też przez cienkie wrappery w analytics/build).
func EvaluateActivities(rw *openrtb_ext.RequestWrapper, ac privacy.ActivityControl, componentName string) (bool, *openrtb_ext.RequestWrapper) {
	component := privacy.Component{Type: privacy.ComponentTypeAnalytics, Name: componentName}
	if !ac.Allow(privacy.ActivityReportAnalytics, component, privacy.ActivityRequest{}) {
		return false, nil
	}
	blockUserFPD := !ac.Allow(privacy.ActivityTransmitUserFPD, component, privacy.ActivityRequest{})
	blockPreciseGeo := !ac.Allow(privacy.ActivityTransmitPreciseGeo, component, privacy.ActivityRequest{})

	if !blockUserFPD && !blockPreciseGeo {
		return true, nil
	}

	cloneReq := &openrtb_ext.RequestWrapper{BidRequest: ortb.CloneBidRequestPartial(rw.BidRequest)}

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

func UpdateReqWrapperForAnalytics(rw *openrtb_ext.RequestWrapper, adapterName string, isCloned bool) *openrtb_ext.RequestWrapper {
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

	if _, ok := reqExtPrebid.Analytics[adapterName]; !ok {
		reqExtPrebid.Analytics = nil
	} else {
		reqExtPrebid.Analytics = UpdatePrebidAnalyticsMap(reqExtPrebid.Analytics, adapterName)
	}
	reqExt.SetPrebid(reqExtPrebid)
	rw.RebuildRequest()

	if cloneReq != nil {
		cloneReq.RebuildRequest()
	}
	return cloneReq
}

func UpdatePrebidAnalyticsMap(extPrebidAnalytics map[string]json.RawMessage, adapterName string) map[string]json.RawMessage {
	newMap := make(map[string]json.RawMessage)
	if val, ok := extPrebidAnalytics[adapterName]; ok {
		newMap[adapterName] = val
	}
	return newMap
}
