package plugin

import (
	"encoding/json"
	"fmt"
)

// General config for all plugins. Different plugins may have different configs structure, but
// two fields are mandatory for all of them:
//  1. `so_path`, a string value telling Prebid-server where to load the shared object.
//  2. `enabled`, a boolean value telling whether or not to load it.
type Config struct {
	SoPath  string `mapstructure:"so_path" json:"so_path"`
	Enabled bool   `mapstructure:"enabled" json:"enabled"`
}

func ParseConfig(name string, cfgData interface{}) (Config, json.RawMessage) {
	data, err := json.Marshal(cfgData)
	if err != nil {
		message := fmt.Sprintf("Failed to marshal config of plugin %s, err: %+v\n", name, err)
		panic(message)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		message := fmt.Sprintf("Config of plugin %s is invalid , err: %+v\n", name, err)
		panic(message)
	}

	return cfg, data
}
