package wurfl_devicedetection

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	defaultCacheSize = "200000"
)

// newConfig creates and validates a new config from the raw JSON data.
func newConfig(data json.RawMessage) (config, error) {
	var cfg config
	if err := jsonutil.UnmarshalValid(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config: %s", err)
	}
	err := cfg.validate()
	return cfg, err
}

// config represents the configuration for the module.
type config struct {
	WURFLSnapshotURL    string   `json:"wurfl_snapshot_url"`    // WURFLSnapshotURL is the WURFL Snapshot URL.
	WURFLFileDirPath    string   `json:"wurfl_file_dir_path"`   // WURFLFileDirPath is the folder where the WURFL data is stored. Required.
	WURFLRunUpdater     *bool    `json:"wurfl_run_updater"`     // WURFLRunUpdater enable the WURFL updater. Default to true
	WURFLCacheSize      int      `json:"wurfl_cache_size"`      // WURFLCacheSize is the size of the WURFL Engine cache. Default is 200000
	AllowedPublisherIDs []string `json:"allowed_publisher_ids"` // Holds the list of allowed publisher IDs. Leave empty to allow all.
	ExtCaps             bool     `json:"ext_caps"`              // ExtCaps if true will include licensed WURFL capabilities in ortb2.Device.Ext
}

// WURFLEngineCacheSize returns the cache size for the WURFL engine.
func (cfg config) WURFLEngineCacheSize() string {
	if cfg.WURFLCacheSize > 0 {
		return strconv.Itoa(cfg.WURFLCacheSize)
	}
	return defaultCacheSize
}

// WURFLFilePath returns the path to the WURFL file.
func (cfg config) WURFLFilePath() string {
	return filepath.Join(cfg.WURFLFileDirPath, filepath.Base(cfg.WURFLSnapshotURL))
}

func (cfg config) validate() error {
	if cfg.WURFLSnapshotURL == "" {
		return fmt.Errorf("wurfl_snapshot_url is required")
	}
	if cfg.WURFLFileDirPath == "" {
		return fmt.Errorf("wurfl_file_dir_path is required")
	}
	return nil
}
