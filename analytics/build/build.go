package build

import (
	"github.com/benbjohnson/clock"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/analytics/clients"
	"github.com/prebid/prebid-server/analytics/filesystem"
	"github.com/prebid/prebid-server/analytics/pubstack"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/privacy"
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
		scopedName := privacy.Component{Type: privacy.ComponentTypeAnalytics, Name: name}
		reportAnalyticsActivityAllowed := ac.Allow(privacy.ActivityReportAnalytics, scopedName)
		if reportAnalyticsActivityAllowed {
			module.LogAuctionObject(ao)
		}
	}
}

func (ea enabledAnalytics) LogVideoObject(vo *analytics.VideoObject, ac privacy.ActivityControl) {
	for name, module := range ea {
		scopedName := privacy.Component{Type: privacy.ComponentTypeAnalytics, Name: name}
		reportAnalyticsActivityAllowed := ac.Allow(privacy.ActivityReportAnalytics, scopedName)
		if reportAnalyticsActivityAllowed {
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
		scopedName := privacy.Component{Type: privacy.ComponentTypeAnalytics, Name: name}
		reportAnalyticsActivityAllowed := ac.Allow(privacy.ActivityReportAnalytics, scopedName)
		if reportAnalyticsActivityAllowed {
			module.LogAmpObject(ao)
		}
	}
}

func (ea enabledAnalytics) LogNotificationEventObject(ne *analytics.NotificationEvent, ac privacy.ActivityControl) {
	for name, module := range ea {
		scopedName := privacy.Component{Type: privacy.ComponentTypeAnalytics, Name: name}
		reportAnalyticsActivityAllowed := ac.Allow(privacy.ActivityReportAnalytics, scopedName)
		if reportAnalyticsActivityAllowed {
			module.LogNotificationEventObject(ne)
		}
	}
}
