package config

import (
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/analytics/filesystem"
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
	return modules
}

//Collection of all the correctly configured analytics modules - implements the PBSAnalyticsModule interface
type enabledAnalytics []analytics.PBSAnalyticsModule

func (ea enabledAnalytics) LogAuctionObject(ao *analytics.AuctionObject) {
	for _, module := range ea {
		module.LogAuctionObject(ao)
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
