package analytics

import (
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
)

type PBSAnalyticsModule interface {
	LogAuctionObject(*AuctionObject)
	LogCookieSyncObject(*CookieSyncObject)
	LogSetUIDObject(*SetUIDObject)
	LogAmpObject(*AmpObject)
}

//Collection of all the correctly configured analytics modules - implements the PBSAnalyticsModule interface
type enabledAnalytics []PBSAnalyticsModule

func NewPBSAnalytics(analytics *config.Analytics) PBSAnalyticsModule {
	modules := make(enabledAnalytics, 0)
	if len(analytics.File.Config) >= 0 {
		if mod, err := NewFileLogger(analytics.File.Config); err == nil {
			modules = append(modules, mod)
		} else {
			glog.Errorf("Could not initialize FileLogger for file %v :%v", analytics.File.Config, err)
		}
	}
	return &modules
}

func (ea *enabledAnalytics) LogAuctionObject(ao *AuctionObject) {
	for _, module := range *ea {
		module.LogAuctionObject(ao)
	}
}

func (ea *enabledAnalytics) LogCookieSyncObject(cso *CookieSyncObject) {
	for _, module := range *ea {
		module.LogCookieSyncObject(cso)
	}
}

func (ea *enabledAnalytics) LogSetUIDObject(so *SetUIDObject) {
	for _, module := range *ea {
		module.LogSetUIDObject(so)
	}
}

func (ea *enabledAnalytics) LogAmpObject(ao *AmpObject) {
	for _, module := range *ea {
		module.LogAmpObject(ao)
	}
}
