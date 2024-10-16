package devicedetection

import (
	"encoding/json"
	"os"

	"github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/pkg/errors"

	"github.com/prebid/prebid-server/v2/util/jsonutil"
)

type config struct {
	DataFile      dataFile      `json:"data_file"`
	AccountFilter accountFilter `json:"account_filter"`
	Performance   performance   `json:"performance"`
}

type dataFile struct {
	Path         string         `json:"path"`
	Update       dataFileUpdate `json:"update"`
	MakeTempCopy *bool          `json:"make_temp_copy"`
}

type dataFileUpdate struct {
	Auto            bool   `json:"auto"`
	Url             string `json:"url"`
	License         string `json:"license_key"`
	PollingInterval int    `json:"polling_interval"`
	Product         string `json:"product"`
	WatchFileSystem *bool  `json:"watch_file_system"`
	OnStartup       bool   `json:"on_startup"`
}

type accountFilter struct {
	AllowList []string `json:"allow_list"`
}

type performance struct {
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

func (c *config) getPerformanceProfile() dd.PerformanceProfile {
	mappedResult, ok := performanceProfileMap[c.Performance.Profile]
	if !ok {
		return dd.Default
	}

	return mappedResult
}

func parseConfig(data json.RawMessage) (config, error) {
	var cfg config
	if err := jsonutil.UnmarshalValid(data, &cfg); err != nil {
		return cfg, errors.Wrap(err, "failed to parse config")
	}
	return cfg, nil
}

func validateConfig(cfg config) error {
	_, err := os.Stat(cfg.DataFile.Path)
	if err != nil {
		return errors.Wrap(err, "error opening hash file path")
	}

	return nil
}
