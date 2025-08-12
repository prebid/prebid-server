package analytics

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
)

//go:generate go run ./generator/buildergen.go

// NewBuilder returns a new analytics module builder.
func NewBuilder() Builder {
	return &builder{builders()}
}

// AnalyticsModuleBuilders: map[vendor]map[module]AnalyticsModuleBuilderFn
type AnalyticsModuleBuilders map[string]map[string]AnalyticsModuleBuilderFn

// AnalyticsModuleBuilderFn – każdy moduł analityczny powinien eksportować symbol `Builder` o tej sygnaturze.
// Zwraca analytics.Module.
type AnalyticsModuleBuilderFn func(cfg json.RawMessage, deps moduledeps.ModuleDeps) (Module, error)

type Builder interface {
	Build(cfg config.Analytics, deps moduledeps.ModuleDeps) (map[string]Module, error)
}

type builder struct {
	builders AnalyticsModuleBuilders
}

func (b *builder) Build(cfg config.Analytics, deps moduledeps.ModuleDeps) (map[string]Module, error) {
	// TODO: włączanie po host/account, flagach enabled itd.

	out := make(map[string]Module)

	for vendor, moduleBuilders := range b.builders {
		for moduleName, buildFn := range moduleBuilders {
			m, err := buildFn(nil, deps)
			if err != nil {
				return nil, err
			}
			out[vendor+"."+moduleName] = m
		}
	}

	return out, nil
}
