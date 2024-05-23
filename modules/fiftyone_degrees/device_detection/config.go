package device_detection

import (
	"encoding/json"
	"github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/pkg/errors"
	"os"

	"github.com/prebid/prebid-server/v2/util/jsonutil"
)

type Config struct {
	DataFile      DataFile      `json:"data_file"`
	AccountFilter AccountFilter `json:"account_filter"`
	Performance   Performance   `json:"performance"`
}

type DataFile struct {
	Path         string         `json:"path"`
	Update       DataFileUpdate `json:"update"`
	MakeTempCopy *bool          `json:"make_temp_copy"`
}

type DataFileUpdate struct {
	Auto            bool   `json:"auto"`
	Url             string `json:"url"`
	License         string `json:"license_key"`
	PollingInterval int    `json:"polling_interval"`
	Product         string `json:"product"`
	WatchFileSystem *bool  `json:"watch_file_system"`
}

type AccountFilter struct {
	AllowList []string `json:"allow_list"`
}

type Performance struct {
	Profile        string `json:"profile"`
	Concurrency    *int   `json:"concurrency"`
	Difference     *int   `json:"difference"`
	AllowUnmatched *bool  `json:"allow_unmatched"`
	Drift          *int   `json:"drift"`
}

var performanceProfileMap = map[string]dd.PerformanceProfile{
	"Default":         dd.Default,
	"LowMemory":       dd.LowMemory,
	"BalancedTemp":    dd.BalancedTemp,
	"Balanced":        dd.Balanced,
	"HighPerformance": dd.HighPerformance,
	"InMemory":        dd.InMemory,
}

func (c *Config) GetPerformanceProfile() dd.PerformanceProfile {
	mappedResult, ok := performanceProfileMap[c.Performance.Profile]
	if !ok {
		return dd.Default
	}

	return mappedResult
}

func ParseConfig(data json.RawMessage) (Config, error) {
	var cfg Config
	if err := jsonutil.UnmarshalValid(data, &cfg); err != nil {
		return cfg, errors.Wrap(err, "failed to parse config")
	}
	return cfg, nil
}

func ValidateConfig(cfg Config) error {
	_, err := os.Stat(cfg.DataFile.Path)
	if err != nil {
		return errors.Wrap(err, "error opening hash file path")
	}

	return nil
}
