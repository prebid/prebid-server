package analytics

import (
	"github.com/benbjohnson/clock"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/analytics"
	moduledeps "github.com/prebid/prebid-server/v3/analytics/analyticsdeps"
	"github.com/prebid/prebid-server/v3/analytics/build"
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

func New(cfg map[string]interface{}) analytics.Runner {
	modules := make(build.EnabledAnalytics)

	deps := moduledeps.Deps{
		HTTPClient: clients.GetDefaultHttpInstance(),
		Clock:      clock.New(),
	}

	for moduleName, buildFn := range analytics.Builders() {
		moduleCfg := cfg[moduleName]
		m, err := buildFn(moduleCfg, deps)
		if err != nil {
			glog.Errorf("Could not initialize analytics module %s: %v", moduleName, err)
			continue
		}
		if m != nil {
			modules[moduleName] = m
		}
	}
	return modules
}
