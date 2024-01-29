package build

import (
	"github.com/benbjohnson/clock"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v2/analytics"
	"github.com/prebid/prebid-server/v2/analytics/clients"
	"github.com/prebid/prebid-server/v2/analytics/filesystem"
	"github.com/prebid/prebid-server/v2/analytics/pubstack"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/ortb"
	"github.com/prebid/prebid-server/v2/privacy"
)

// Modules that need to be logged to need to be initialized here
func New(analytics *config.Analytics) analytics.Runner {
	modules := make(enabledAnalytics, 0)
	if len(analytics.File.Filename) > 0 {
		if mod, err := filesystem.NewFileLogger(analytics.File.Filename); err == nil {
			modules["filelogger"] = mod
		} else {
			glog.Fatalf("Could not initialize FileLogger for file %v :%v", analytics.File.Filename, err)
		}
	}

	if analytics.Pubstack.Enabled {
		pubstackModule, err := pubstack.NewModule(
			clients.GetDefaultHttpInstance(),
			analytics.Pubstack.ScopeId,
			analytics.Pubstack.IntakeUrl,
			analytics.Pubstack.ConfRefresh,
			analytics.Pubstack.Buffers.EventCount,
			analytics.Pubstack.Buffers.BufferSize,
			analytics.Pubstack.Buffers.Timeout,
			clock.New())
		if err == nil {
			modules["pubstack"] = pubstackModule
		} else {
			glog.Errorf("Could not initialize PubstackModule: %v", err)
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
			module.LogAuctionObject(ao)
		}
	}
}

func (ea enabledAnalytics) LogVideoObject(vo *analytics.VideoObject, ac privacy.ActivityControl) {
	for name, module := range ea {
		if isAllowed, cloneBidderReq := evaluateActivities(vo.RequestWrapper, ac, name); isAllowed {
			if cloneBidderReq != nil {
				vo.RequestWrapper = cloneBidderReq
			}
			module.LogVideoObject(vo)
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
			module.LogAmpObject(ao)
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
