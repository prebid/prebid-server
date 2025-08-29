package analytics

import (
	"github.com/benbjohnson/clock"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/analytics/clients"
	"github.com/prebid/prebid-server/v3/analytics/moduledeps"
)

//go:generate go run ./generator/buildergen.go

// NewBuilder returns a new analytics module builder.
func NewBuilder() Builder {
	return &builder{builders()}
}

// AnalyticsModuleBuilders: map[vendor]map[module]AnalyticsModuleBuilderFn
type AnalyticsModuleBuilders map[string]map[string]AnalyticsModuleBuilderFn

type AnalyticsModuleBuilderFn func(cfg map[string]interface{}, deps moduledeps.ModuleDeps) (Module, error)

func New(cfg map[string]interface{}) analytics.Runner {
	modules := make(enabledAnalytics)

	deps := moduledeps.ModuleDeps{
		HTTPClient: clients.GetDefaultHttpInstance(),
		Clock:      clock.New(),
	}

	for vendor, moduleBuilders := range analytics.Builders() {
		for moduleName, buildFn := range moduleBuilders {
			dir := configDir(cfg, vendor, moduleName)
			m, err := buildFn(dir, deps)
			if err != nil {
				glog.Errorf("Could not initialize analytics module %s.%s: %v", vendor, moduleName, err)
				continue
			}
			if m != nil {
				modules[moduleName] = m
			}
		}
	}
	return modules
}

func configDir(root map[string]interface{}, vendor, module string) map[string]interface{} {
	if root == nil {
		return nil
	}
	if m, ok := root[module].(map[string]interface{}); ok {
		return m
	}
	if v, ok := root[vendor].(map[string]interface{}); ok {
		if m, ok := v[module].(map[string]interface{}); ok {
			return m
		}
	}

	return map[string]interface{}{}
}
