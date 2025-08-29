package filelogger

import (
	"github.com/mitchellh/mapstructure"

	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
)

// Config is the minimal configuration for the file logger analytics module.
// Empty Filename means the module is disabled.
type Config struct {
	Enabled  bool   `mapstructure:"enabled" json:"enabled"`
	Filename string `mapstructure:"filename" json:"filename"`
}

// Builder builds the filelogger analytics module.
func Builder(cfg map[string]interface{}, deps moduledeps.ModuleDeps) (analytics.Module, error) {
	var c Config
	if cfg != nil {
		if err := mapstructure.Decode(cfg, &c); err != nil {
			return nil, err
		}
	}

	// Disabled if filename is empty
	if c.Filename == "" {
		return nil, nil
	}

	if !c.Enabled {
		return nil, nil
	}
	return NewFileLogger(c.Filename)
}
