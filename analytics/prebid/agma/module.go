package agma

import (
	"github.com/benbjohnson/clock"
	"github.com/mitchellh/mapstructure"
	"github.com/prebid/prebid-server/v3/analytics"
	base "github.com/prebid/prebid-server/v3/analytics/agma"
	"github.com/prebid/prebid-server/v3/analytics/moduledeps"
	"github.com/prebid/prebid-server/v3/config"
)

type Config struct {
	Enabled  bool   `mapstructure:"enabled" json:"enabled"`
	Endpoint string `mapstructure:"endpoint" json:"endpoint"`
	Buffers  struct {
		EventCount int    `mapstructure:"eventCount" json:"eventCount"`
		BufferSize string `mapstructure:"bufferSize" json:"bufferSize"`
		Timeout    string `mapstructure:"timeout" json:"timeout"`
	} `mapstructure:"buffers" json:"buffers"`
	Accounts []config.AgmaAnalyticsAccount `mapstructure:"accounts" json:"accounts"`
}

// Builder builds the agma analytics module.
func Builder(cfg map[string]interface{}, deps moduledeps.ModuleDeps) (analytics.Module, error) {
	if deps.HTTPClient == nil || deps.Clock == nil {
		return nil, nil
	}

	var c Config
	if cfg != nil {
		if err := mapstructure.Decode(cfg, &c); err != nil {
			return nil, err
		}
	}

	if !c.Enabled {
		return nil, nil
	}
	if c.Endpoint == "" {
		return nil, nil
	}

	full := config.AgmaAnalytics{
		Enabled:  true,
		Endpoint: c.Endpoint,
		Buffers: config.AgmaAnalyticsBuffers{
			EventCount: c.Buffers.EventCount,
			BufferSize: c.Buffers.BufferSize,
			Timeout:    c.Buffers.Timeout,
		},
		Accounts: c.Accounts,
	}
	return base.NewModule(deps.HTTPClient, full, deps.Clock.(clock.Clock))
}
