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
	Path   string         `json:"path"`
	Update DataFileUpdate `json:"update"`
}

type DataFileUpdate struct {
	Auto            bool   `json:"auto"`
	Url             string `json:"url"`
	License         string `json:"license_key"`
	PollingInterval int    `json:"polling_interval"`
	Product         string `json:"product"`
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
	"default":          dd.Default,
	"low_memory":       dd.LowMemory,
	"balanced_temp":    dd.BalancedTemp,
	"balanced":         dd.Balanced,
	"high_performance": dd.HighPerformance,
	"in_memory":        dd.InMemory,
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
