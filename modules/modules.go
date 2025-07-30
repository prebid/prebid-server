package modules

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/hooks"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

//go:generate go run ./generator/buildergen.go

// NewBuilder returns a new module builder.
func NewBuilder() Builder {
	return &builder{builders()}
}

// Builder is the interfaces intended for building modules
// implementing hook interfaces [github.com/prebid/prebid-server/hooks/hookstage].
type Builder interface {
	// Build initializes existing hook modules passing them config and other dependencies.
	// It returns hook repository created based on the implemented hook interfaces by modules
	// and a map of modules to a list of stage names for which module provides hooks
	// or an error encountered during module initialization.
	Build(cfg config.Modules, client moduledeps.ModuleDeps) (hooks.HookRepository, map[string][]string, *ShutdownModules, error)
}

type (
	// ModuleBuilders mapping between module name and its builder: map[vendor]map[module]ModuleBuilderFn
	ModuleBuilders map[string]map[string]ModuleBuilderFn
	// ModuleBuilderFn returns an interface{} type that implements certain hook interfaces.
	ModuleBuilderFn func(cfg json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error)
)

type builder struct {
	builders ModuleBuilders
}

// Build walks over the list of registered modules and initializes them.
//
// The ID chosen for the module's hooks represents a fully qualified module path in the format
// "vendor.module_name" and should be used to retrieve module hooks from the hooks.HookRepository.
//
// Method returns a hooks.HookRepository and a map of modules to a list of stage names
// for which module provides hooks or an error occurred during modules initialization.
func (m *builder) Build(
	cfg config.Modules,
	deps moduledeps.ModuleDeps,
) (hooks.HookRepository, map[string][]string, *ShutdownModules, error) {
	modules := make(map[string]interface{})
	for vendor, moduleBuilders := range m.builders {
		for moduleName, builder := range moduleBuilders {
			var err error
			var conf json.RawMessage
			var isEnabled bool

			id := fmt.Sprintf("%s.%s", vendor, moduleName)
			if data, ok := cfg[vendor][moduleName]; ok {
				if conf, err = jsonutil.Marshal(data); err != nil {
					return nil, nil, nil, fmt.Errorf(`failed to marshal "%s" module config: %s`, id, err)
				}

				if values, ok := data.(map[string]interface{}); ok {
					if value, ok := values["enabled"].(bool); ok {
						isEnabled = value
					}
				}
			}

			if !isEnabled {
				glog.Infof("Skip %s module, disabled.", id)
				continue
			}

			module, err := builder(conf, deps)
			if err != nil {
				return nil, nil, nil, fmt.Errorf(`failed to init "%s" module: %s`, id, err)
			}

			modules[id] = module
		}
	}

	collection, err := createModuleStageNamesCollection(modules)
	if err != nil {
		return nil, nil, nil, err
	}

	repo, err := hooks.NewHookRepository(modules)

	sdm := NewShutdownModules(modules)

	return repo, collection, sdm, err
}
