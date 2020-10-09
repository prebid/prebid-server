package config

import (
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/analytics/clients"
	"github.com/prebid/prebid-server/analytics/filesystem"
	"github.com/prebid/prebid-server/analytics/pubstack"
	"github.com/prebid/prebid-server/config"
)

//Modules that need to be logged to need to be initialized here
func NewPBSAnalytics(analytics *config.Analytics) analytics.PBSAnalyticsModule {
	modules := make(enabledAnalytics, 0)
	if len(analytics.File.Filename) > 0 {
		if mod, err := filesystem.NewFileLogger(analytics.File.Filename); err == nil {
			modules = append(modules, mod)
		} else {
			glog.Fatalf("Could not initialize FileLogger for file %v :%v", analytics.File.Filename, err)
		}
	}
	if analytics.Pubstack.Enabled {
		pubstackModule, err := pubstack.NewPubstackModule(
			clients.GetDefaultHttpInstance(),
			analytics.Pubstack.ScopeId,
			analytics.Pubstack.IntakeUrl,
			analytics.Pubstack.ConfRefresh,
			analytics.Pubstack.Buffers.EventCount,
			analytics.Pubstack.Buffers.BufferSize,
			analytics.Pubstack.Buffers.Timeout)
		if err == nil {
			modules = append(modules, pubstackModule)
		} else {
			glog.Errorf("Could not initialize PubstackModule: %v", err)
		}
	}
	return modules
}

//Collection of all the correctly configured analytics modules - implements the PBSAnalyticsModule interface
type enabledAnalytics []analytics.PBSAnalyticsModule

func (ea enabledAnalytics) LogAuctionObject(ao *analytics.AuctionObject) {
	for _, module := range ea {
		module.LogAuctionObject(ao)
	}
}

func (ea enabledAnalytics) LogVideoObject(vo *analytics.VideoObject) {
	for _, module := range ea {
		module.LogVideoObject(vo)
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

func (ea enabledAnalytics) LogAmpObject(ao *analytics.AmpObject) {
	for _, module := range ea {
		module.LogAmpObject(ao)
	}
}

func (ea enabledAnalytics) LogNotificationEventObject(ne *analytics.NotificationEvent) {
	for _, module := range ea {
		module.LogNotificationEventObject(ne)
	}
}
