package analytics

import (
	"github.com/benbjohnson/clock"
	"github.com/golang/glog"
	moduledeps "github.com/prebid/prebid-server/v3/analytics/analyticsdeps"
	"github.com/prebid/prebid-server/v3/analytics/clients"
)

//go:generate go run ./generator/buildergen.go

// NewBuilder returns a new analytics module builder.
func NewBuilder() AnalyticsModuleBuilders {
	return builders()
}

// AnalyticsModuleBuilders: map[vendor]map[module]AnalyticsModuleBuilderFn
type AnalyticsModuleBuilders map[string]map[string]AnalyticsModuleBuilderFn

type AnalyticsModuleBuilderFn func(cfg map[string]interface{}, deps moduledeps.Deps) (Module, error)

func New(cfg map[string]interface{}) Runner {
	modules := make(EnabledAnalytics)

	deps := moduledeps.Deps{
		HTTPClient: clients.GetDefaultHttpInstance(),
		Clock:      clock.New(),
	}

	for _, vendorBuilders := range Builders() {
		for moduleName, buildFn := range vendorBuilders {
			var moduleCfg map[string]interface{}
			if v, ok := cfg[moduleName].(map[string]interface{}); ok {
				moduleCfg = v
			}
			m, err := buildFn(moduleCfg, deps)
			if err != nil {
				glog.Errorf("Could not initialize analytics module %s: %v", moduleName, err)
				continue
			}
			if m != nil {
				modules[moduleName] = m
			}
		}
	}
	return modules
}
