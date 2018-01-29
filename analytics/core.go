package analytics

import (
	"log"
	"github.com/golang/glog"
)

type PBSAnalyticsModule interface {
	LogAuctionObject(*AuctionObject)
	LogCookieSyncObject(*CookieSyncObject)
	LogSetUIDObject(*SetUIDObject)
}

//Collection of all the correctly configured analytics modules - implements the PBSAnalyticsModule interface
type enabledAnalytics []PBSAnalyticsModule

type factory func(conf map[string]string) (PBSAnalyticsModule, error)

var analyticsFactories = make(map[string]factory)

//Assign factory method to respective modules
func Register(name string, factory factory) {
	if factory == nil {
		log.Panicf("Analytics factory %s does not exist.", name)
	}
	_, registered := analyticsFactories[name]
	if registered {
		glog.Errorf("Analytics factory %s already registered. Ignoring.", name)
	}
	analyticsFactories[name] = factory
}

//Setup and initialize analytics modules
func CreateAnalyticsModules(conf map[string]string) (PBSAnalyticsModule) {
	modules := make(enabledAnalytics, 0)
	for module := range conf {
		engineName, ok := conf[module]
		if ok {
			engineFactory, ok := analyticsFactories[engineName]
			if ok {
				if mod, err := engineFactory(conf); err == nil {
					modules = append(modules, mod)
				} else {
					glog.Errorf("Error setting up %v", module)
				}
			} else {
				glog.Errorf("Factory missing for module %v", module)
			}
		} else {
			glog.Error("Module configuration not found for %v", module)
		}
	}
	return modules
}

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
