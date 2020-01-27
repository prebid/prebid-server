package config

import (
	"encoding/json"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/analytics/filesystem"
	"github.com/prebid/prebid-server/analytics/newsiq"
	"github.com/prebid/prebid-server/config"
)

//Modules that need to be logged to need to be initialized here
func NewPBSAnalytics(analytics *config.Analytics) analytics.PBSAnalyticsModule {
	println("Testing NewPBSAnalytics")
	modules := make(enabledAnalytics, 0)
	// println("Enabled: ", enabledAnalytics)
	println("Modules: ", modules)
	if len(analytics.File.Filename) > 0 {
		// if mod, err := filesystem.NewFileLogger(analytics.File.Filename); err == nil { // OLD
		if mod, err := filesystem.NewFileLogger(""); err == nil { // NEW
			modules = append(modules, mod)
		} else {
			glog.Fatalf("Could not initialize FileLogger for file %v :%v", analytics.File.Filename, err)
		}
	}

	dataLogger := newsiq.NewDataLogger("TestNewsIQDataLogger")
	modules = append(modules, dataLogger)
	return modules
}

//Collection of all the correctly configured analytics modules - implements the PBSAnalyticsModule interface
type enabledAnalytics []analytics.PBSAnalyticsModule

func (ea enabledAnalytics) LogAuctionObject(ao *analytics.AuctionObject) {

	println("Testing LogAuctionObject Status: ", ao.Status)
	println("Testing LogAuctionObject Errors: ", ao.Errors)

	reqout, err := json.Marshal(ao.Request)
	if err != nil {
		panic(err)
	}
	println("Testing LogAuctionObject Request: ", string(reqout))
	println("Testing LogAuctionObject Request: ", ao.Request)

	resout, err := json.Marshal(ao.Response)
	if err != nil {
		panic(err)
	}
	println("Testing LogAuctionObject Response: ", string(resout))
	println("Testing LogAuctionObject Response: ", ao.Response)
	for _, module := range ea {
		module.LogAuctionObject(ao)
	}
}

func (ea enabledAnalytics) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	out, err := json.Marshal(cso)
	if err != nil {
		panic(err)
	}
	println("Testing LogCookieSyncObject Response: ", string(out))
	println("Testing LogCookieSyncObject Response: ", cso)

	for _, module := range ea {
		module.LogCookieSyncObject(cso)
	}
}

func (ea enabledAnalytics) LogSetUIDObject(so *analytics.SetUIDObject) {
	out, err := json.Marshal(so)
	if err != nil {
		panic(err)
	}
	println("Testing LogSetUIDObject Response: ", string(out))
	println("Testing LogSetUIDObject Response: ", so)
	for _, module := range ea {
		module.LogSetUIDObject(so)
	}
}

func (ea enabledAnalytics) LogAmpObject(ao *analytics.AmpObject) {
	println("Testing LogAmpObject")
	for _, module := range ea {
		module.LogAmpObject(ao)
	}
}
