package wurfl_devicedetection

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	defaultCacheSize = "200000"
)

var ErrWURFLFilePathRequired = errors.New("wurfl_file_path is required")

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
	// WURFLFilePath is the path to the WURFL file (i.e. /path/to/wurfl.zip). Required.
	WURFLFilePath string `json:"wurfl_file_path"`
	// WURFLSnapshotURL is the URL of the WURFL Snapshot.
	// If set, it will be used to periodically update the WURFL file.
	// The snapshot will be downloaded to the same directory as the WURFLFilePath.
	// Make sure this directory is writable.
	WURFLSnapshotURL string `json:"wurfl_snapshot_url"`
	// WURFLCacheSize is the size of the WURFL Engine cache. Default is 200000
	WURFLCacheSize int `json:"wurfl_cache_size"`
	// Holds the list of allowed publisher IDs. Leave empty to allow all.
	AllowedPublisherIDs []string `json:"allowed_publisher_ids"`
	// ExtCaps if true will include licensed WURFL capabilities in ortb2.Device.Ext
	ExtCaps bool `json:"ext_caps"`
}

// WURFLEngineCacheSize returns the cache size for the WURFL engine.
func (cfg config) WURFLEngineCacheSize() string {
	if cfg.WURFLCacheSize > 0 {
		return strconv.Itoa(cfg.WURFLCacheSize)
	}
	return defaultCacheSize
}

func (cfg config) validate() error {
	if cfg.WURFLFilePath == "" {
		return ErrWURFLFilePathRequired
	}
	return nil
}
