package agma

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/analytics/analyticsdeps"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type Config struct {
	Enabled  bool                             `"enabled" json:"enabled"`
	Endpoint config.AgmaAnalyticsHttpEndpoint `"endpoint" json:"endpoint"`
	Buffers  struct {
		EventCount int    `"eventCount" json:"eventCount"`
		BufferSize string `"bufferSize" json:"bufferSize"`
		Timeout    string `"timeout" json:"timeout"`
	} `"buffers" json:"buffers"`
	Accounts []config.AgmaAnalyticsAccount `"accounts" json:"accounts"`
}

// Builder builds the agma analytics module.
func Builder(cfg json.RawMessage, deps analyticsdeps.Deps) (analytics.Module, error) {
	if deps.HTTPClient == nil || deps.Clock == nil {
		return nil, nil
	}

	var c Config
	if cfg != nil {
		if err := jsonutil.Unmarshal(cfg, &c); err != nil {
			return nil, err
		}
	}

	if !c.Enabled {
		return nil, nil
	}
	if c.Endpoint.Url == "" {
		return nil, nil
	}

	full := config.AgmaAnalytics{
		Enabled:  true,
		Endpoint: c.Endpoint,
		Buffers: config.AgmaAnalyticsBuffer{
			EventCount: c.Buffers.EventCount,
			BufferSize: c.Buffers.BufferSize,
			Timeout:    c.Buffers.Timeout,
		},
		Accounts: c.Accounts,
	}
	return NewModule(deps.HTTPClient, full, deps.Clock)
}
