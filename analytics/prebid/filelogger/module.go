package filelogger

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/analytics/filesystem"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
)

// Config is the minimal configuration for the file logger analytics module.
// It currently matches the existing global config field analytics.file.filename
// but is structured to allow future per-module config.
type Config struct {
	Filename string `json:"filename"`
}

// Builder builds the filelogger analytics module.
func Builder(raw json.RawMessage, deps moduledeps.ModuleDeps) (analytics.Module, error) {
	var cfg Config
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, err
		}
	}
	if cfg.Filename == "" {
		// fallback: attempt to use global config via deps if ever added; for now, no module if empty
		return nil, nil
	}
	return filesystem.NewFileLogger(cfg.Filename)
}
