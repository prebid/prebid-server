package analytics

import (
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
)

/*
  	PBSAnalyticsModule must be implemented by any analytics module that does transactional logging.

	New modules can use the /analytics/endpoint_data_objects, extract the
	information required and are responsible for handling all their logging activities inside LogAuctionObject, LogAmpObject
	LogCookieSyncObject and LogSetUIDObject method implementations.
*/

type PBSAnalyticsModule interface {
	LogAuctionObject(*AuctionObject)
	LogCookieSyncObject(*CookieSyncObject)
	LogSetUIDObject(*SetUIDObject)
	LogAmpObject(*AmpObject)
}

//Collection of all the correctly configured analytics modules - implements the PBSAnalyticsModule interface
type enabledAnalytics []PBSAnalyticsModule

//Modules that need to be logged to need to be initialized here
func NewPBSAnalytics(analytics *config.Analytics) PBSAnalyticsModule {
	modules := make(enabledAnalytics, 0)
	if len(analytics.File.Filename) > 0 {
		if mod, err := NewFileLogger(analytics.File.Filename); err == nil {
			modules = append(modules, mod)
		} else {
			glog.Fatalf("Could not initialize FileLogger for file %v :%v", analytics.File.Filename, err)
		}
	}
	return modules
}

/*
	This could be confusing. `enabledAnalytics` itself implements `PBSAnalyticsModule` as well wherein it iterates through each analytic module and calls it's respective `Log{loggable_object}Object` method.
*/

func (ea enabledAnalytics) LogAuctionObject(ao *AuctionObject) {
	for _, module := range ea {
		module.LogAuctionObject(ao)
	}
}

func (ea enabledAnalytics) LogCookieSyncObject(cso *CookieSyncObject) {
	for _, module := range ea {
		module.LogCookieSyncObject(cso)
	}
}

func (ea enabledAnalytics) LogSetUIDObject(so *SetUIDObject) {
	for _, module := range ea {
		module.LogSetUIDObject(so)
	}
}

func (ea enabledAnalytics) LogAmpObject(ao *AmpObject) {
	for _, module := range ea {
		module.LogAmpObject(ao)
	}
}
