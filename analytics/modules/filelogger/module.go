package filelogger

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v3/util/jsonutil"

	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/analytics/analyticsdeps"
)

// Config is the minimal configuration for the file logger analytics module.
// Empty Filename means the module is disabled.
type Config struct {
	Enabled  bool   `json:"enabled"`
	Filename string `json:"filename"`
}

// Builder builds the filelogger analytics module.
func Builder(cfg json.RawMessage, deps analyticsdeps.Deps) (analytics.Module, error) {
	var c Config
	if cfg != nil {
		if err := jsonutil.Unmarshal(cfg, &c); err != nil {
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
