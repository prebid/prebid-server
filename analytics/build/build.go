package build

import (
	"encoding/json"

	"github.com/benbjohnson/clock"
	"github.com/prebid/prebid-server/v4/analytics"
	"github.com/prebid/prebid-server/v4/analytics/agma"
	"github.com/prebid/prebid-server/v4/analytics/clients"
	"github.com/prebid/prebid-server/v4/analytics/filesystem"
	"github.com/prebid/prebid-server/v4/analytics/pubstack"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/logger"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/ortb"
	"github.com/prebid/prebid-server/v4/privacy"
)

// Modules that need to be logged to need to be initialized here
func New(analytics *config.Analytics) analytics.Runner {
	modules := make(enabledAnalytics, 0)
	if len(analytics.File.Filename) > 0 {
		if mod, err := filesystem.NewFileLogger(analytics.File.Filename); err == nil {
			modules["filelogger"] = mod
		} else {
			logger.Fatalf("Could not initialize FileLogger for file %v :%v", analytics.File.Filename, err)
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
			logger.Errorf("Could not initialize PubstackModule: %v", err)
		}
	}

	if analytics.Agma.Enabled {
		agmaModule, err := agma.NewModule(
			clients.GetDefaultHttpInstance(),
			analytics.Agma,
			clock.New())
		if err == nil {
			modules["agma"] = agmaModule
		} else {
			logger.Errorf("Could not initialize Agma Anayltics: %v", err)
		}
	}

	return modules
}

// Collection of all the correctly configured analytics modules - implements the PBSAnalyticsModule interface
type enabledAnalytics map[string]analytics.Module

func (ea enabledAnalytics) LogAuctionObject(ao *analytics.AuctionObject, ac privacy.ActivityControl) {
	for name, module := range ea {
		request, allowed := requestForAnalytics(ao.RequestWrapper, ac, name)
		if allowed {
			event := *ao
			event.RequestWrapper = request
			module.LogAuctionObject(&event)
		}
	}
}

func (ea enabledAnalytics) LogVideoObject(vo *analytics.VideoObject, ac privacy.ActivityControl) {
	for name, module := range ea {
		request, allowed := requestForAnalytics(vo.RequestWrapper, ac, name)
		if allowed {
			event := *vo
			event.RequestWrapper = request
			module.LogVideoObject(&event)
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
		request, allowed := requestForAnalytics(ao.RequestWrapper, ac, name)
		if allowed {
			event := *ao
			event.RequestWrapper = request
			module.LogAmpObject(&event)
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

// requestForAnalytics derives one module's view from the incoming request. Clone only
// when activity controls or adapter-specific configuration require a different view.
func requestForAnalytics(rw *openrtb_ext.RequestWrapper, ac privacy.ActivityControl, name string) (*openrtb_ext.RequestWrapper, bool) {
	component := privacy.Component{Type: privacy.ComponentTypeAnalytics, Name: name}
	if !ac.Allow(privacy.ActivityReportAnalytics, component, privacy.ActivityRequest{}) {
		return nil, false
	}
	if rw == nil || rw.BidRequest == nil {
		return rw, true
	}

	blockUserFPD := !ac.Allow(privacy.ActivityTransmitUserFPD, component, privacy.ActivityRequest{})
	blockPreciseGeo := !ac.Allow(privacy.ActivityTransmitPreciseGeo, component, privacy.ActivityRequest{})
	ext, _ := rw.GetRequestExt()
	prebid := ext.GetPrebid()
	filterAnalytics := prebid != nil && len(prebid.Analytics) > 0
	if !blockUserFPD && !blockPreciseGeo && !filterAnalytics {
		return rw, true
	}

	request := &openrtb_ext.RequestWrapper{BidRequest: ortb.CloneBidRequestPartial(rw.BidRequest)}
	if blockUserFPD {
		privacy.ScrubUserFPD(request)
	}
	if blockPreciseGeo {
		privacy.ScrubGeoAndDeviceIP(request, privacy.IPConf{IPV6: ac.IPv6Config, IPV4: ac.IPv4Config})
	}
	if filterAnalytics {
		if config, ok := prebid.Analytics[name]; ok {
			prebid.Analytics = map[string]json.RawMessage{name: config}
		} else {
			prebid.Analytics = nil
		}
		requestExt, _ := request.GetRequestExt()
		requestExt.SetPrebid(prebid)
	}
	request.RebuildRequest()
	return request, true
}
