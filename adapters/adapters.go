package adapters

import (
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbs"
)

// all is unexported because it may contain adapters that have not been configured
var all map[string]pbs.Adapter

// Active allows us to access all of the active adapters that have been configured
var Active map[string]pbs.Adapter

func init() {
	if Active == nil {
		Active = map[string]pbs.Adapter{}
	}
	if all == nil {
		all = map[string]pbs.Adapter{}
	}
}

// Init is called by each adapter so they can be registered in the global map
func Init(name string, ex pbs.Adapter) {
	all[name] = ex
}

// Get will return back an adapter.
// If the adapter has not been registered we will return a false boolean
// example:
// appnexus, ok := adapters.Get("appnexus")
func Get(name string) (pbs.Adapter, bool) {
	if ex, ok := Active[name]; ok {
		return ex, true
	}
	return nil, false
}

// Configure should be called once.
func Configure(externalURL string, cfgs map[string]config.Adapter) error {

	for exchange, cfg := range cfgs {
		ex, ok := all[exchange]
		if !ok {
			// TODO: should return the error below or just log? Right now we are just logging the error and not making the adapter accessible
			glog.Infof("Could not configure exchange: %v", exchange)
			continue
		}
		ex.Configure(externalURL, &cfg)
		Active[exchange] = ex // attach the adapter to the global Active map
	}
	return nil
}
