package modules

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/modules/moduledeps"
	mspPlugin "github.com/prebid/prebid-server/msp/plugin"
)

type PluginBuilder interface {
	Build(json.RawMessage, moduledeps.ModuleDeps) (interface{}, error)
}

func mspLoadModulePlugins(modules map[string]interface{}, cfg config.Modules, deps moduledeps.ModuleDeps) map[string]interface{} {
	for vendor, moduleBuilders := range cfg {
		for moduleName, moduleCfg := range moduleBuilders {
			id := fmt.Sprintf("%s.%s", vendor, moduleName)

			if _, ok := modules[id]; ok {
				// skip loading modules that have already been loaded through hardcoded builder
				continue
			}

			builder, cfgJson, skip, err := mspPlugin.LoadBuilder[PluginBuilder](id, moduleCfg)

			if skip {
				continue
			}

			if err != nil {
				panic(err)
			}

			module, err := builder.Build(cfgJson, deps)
			if err != nil {
				panic(fmt.Sprintf("Failed to build Module plugin %s, error: %+v\n", id, err))
			}

			modules[id] = module
			glog.Infof("Loaded Module plugin %s.\n", id)
		}
	}

	return modules
}
