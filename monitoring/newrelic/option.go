package newrelic

import nr "github.com/newrelic/go-agent/v3/newrelic"

// ConfigIgnoreStatusCodes appends codes to the IgnoreStatusCodes.
func ConfigIgnoreStatusCodes(ignoreStatusCodes []int) nr.ConfigOption {
	return func(cfg *nr.Config) {
		cfg.ErrorCollector.IgnoreStatusCodes = append(cfg.ErrorCollector.IgnoreStatusCodes, ignoreStatusCodes...)
	}
}
