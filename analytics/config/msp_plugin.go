package config

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/analytics"
	mspPlugin "github.com/prebid/prebid-server/msp/plugin"
)

type PluginBuilder interface {
	Build(json.RawMessage) (analytics.PBSAnalyticsModule, error)
}

func mspLoadAnalyticsAdapterPlugins(cfg map[string]interface{}) []analytics.PBSAnalyticsModule {
	plugins := make([]analytics.PBSAnalyticsModule, 0)

	for name, cfgData := range cfg {
		builder, cfgJson, skip, err := mspPlugin.LoadBuilder[PluginBuilder](name, cfgData)

		if skip {
			continue
		}

		if err != nil {
			panic(err)
		}

		plugin, err := builder.Build(cfgJson)
		if err != nil {
			panic(fmt.Sprintf("Failed to build Analytics Adapter plugin %s, error: %+v\n", name, err))
		} else {
			glog.Infof("Loaded Analytics Adapter plugin: %s\n", name)
			plugins = append(plugins, plugin)
		}
	}

	return plugins
}
