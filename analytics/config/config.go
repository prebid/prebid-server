package config

import (
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/analytics/clients"
	"github.com/prebid/prebid-server/analytics/filesystem"
	"github.com/prebid/prebid-server/analytics/pubstack"
	"github.com/prebid/prebid-server/config"
)

type analyticsModule analytics.PBSAnalyticsModule

type pbsAnalyticsModule struct {
	enabledModules []analyticsModule
}

//Modules that need to be logged to need to be initialized here
func NewPBSAnalytics(analytics *config.Analytics) analytics.PBSAnalyticsModule {

	instance := &pbsAnalyticsModule{enabledModules: make([]analyticsModule, 0)}

	if len(analytics.File.Filename) > 0 {
		if mod, err := filesystem.NewFileLogger(analytics.File.Filename); err == nil {
			instance.enabledModules = append(instance.enabledModules, mod)
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
			instance.enabledModules = append(instance.enabledModules, pubstackModule)
		} else {
			glog.Fatalf("Could not initialize PubstackModule: %v", err)
		}
	}
	return instance
}

func (pam pbsAnalyticsModule) LogAuctionObject(ao *analytics.AuctionObject) {
	for _, module := range pam.enabledModules {
		module.LogAuctionObject(ao)
	}
}

func (pam pbsAnalyticsModule) LogVideoObject(vo *analytics.VideoObject) {
	for _, module := range pam.enabledModules {
		module.LogVideoObject(vo)
	}
}

func (pam pbsAnalyticsModule) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	for _, module := range pam.enabledModules {
		module.LogCookieSyncObject(cso)
	}
}

func (pam pbsAnalyticsModule) LogSetUIDObject(so *analytics.SetUIDObject) {
	for _, module := range pam.enabledModules {
		module.LogSetUIDObject(so)
	}
}

func (pam pbsAnalyticsModule) LogAmpObject(ao *analytics.AmpObject) {
	for _, module := range pam.enabledModules {
		module.LogAmpObject(ao)
	}
}
