package modules

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/modules/moduledeps"
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
	Build(cfg config.Modules, client moduledeps.ModuleDeps) (hooks.HookRepository, map[string][]string, error)
}

type (
	// ModuleBuilders mapping between module name and its builder: map[vendor]map[module]ModuleBuilderFn
	ModuleBuilders map[string]map[string]ModuleBuilderFn
	// ModuleBuilderFn returns an interface{} type that implements certain hook interfaces.
	ModuleBuilderFn func(cfg json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error)
	// moduleConfig defines the minimum module config, each module is required to implement it.
	moduleConfig struct {
		// Enabled determines the module status. All modules are disabled by default
		// unless explicitly enabled via config. Disabled modules not added to hook repository
		// and skipped from general hook execution flow. PBS logs warning if hook execution plan
		// refers to disabled module.
		Enabled bool `json:"enabled"`
	}
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
) (hooks.HookRepository, map[string][]string, error) {
	modules := make(map[string]interface{})
	for vendor, moduleBuilders := range m.builders {
		for moduleName, builder := range moduleBuilders {
			var err error
			var conf json.RawMessage
			var baseConf moduleConfig

			id := fmt.Sprintf("%s.%s", vendor, moduleName)
			if data, ok := cfg[vendor][moduleName]; ok {
				if conf, err = json.Marshal(data); err != nil {
					return nil, nil, fmt.Errorf(`failed to marshal "%s" module config: %s`, id, err)
				}
			}

			if conf != nil {
				if err = json.Unmarshal(conf, &baseConf); err != nil {
					return nil, nil, fmt.Errorf(`failed to unmarshal base config for module %s: %s`, id, err)
				}
			}

			if !baseConf.Enabled {
				glog.Infof("Skip %s module, disabled.", id)
				continue
			}

			module, err := builder(conf, deps)
			if err != nil {
				return nil, nil, fmt.Errorf(`failed to init "%s" module: %s`, id, err)
			}

			modules[id] = module
		}
	}

	collection, err := createModuleStageNamesCollection(modules)
	if err != nil {
		return nil, nil, err
	}

	repo, err := hooks.NewHookRepository(modules)

	return repo, collection, err
}
