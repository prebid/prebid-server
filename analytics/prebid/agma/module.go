package agma

import (
	"encoding/json"

	"github.com/benbjohnson/clock"
	"github.com/prebid/prebid-server/v3/analytics"
	base "github.com/prebid/prebid-server/v3/analytics/agma"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
)

// Config reuses config.AgmaAnalytics structure fields needed for module construction.
type Config struct {
	Endpoint string `json:"endpoint"`
	Buffers  struct {
		EventCount int    `json:"eventCount"`
		BufferSize string `json:"bufferSize"`
		Timeout    string `json:"timeout"`
	} `json:"buffers"`
	Accounts []config.AgmaAnalyticsAccount `json:"accounts"`
}

// Builder builds the agma analytics module.
func Builder(raw json.RawMessage, deps moduledeps.ModuleDeps) (analytics.Module, error) {
	if deps.HTTPClient == nil || deps.Clock == nil {
		return nil, nil
	}
	var cfg Config
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, err
		}
	}
	if cfg.Endpoint == "" || len(cfg.Accounts) == 0 {
		return nil, nil
	}
	full := config.AgmaAnalytics{
		Enabled:  true,
		Endpoint: cfg.Endpoint,
		Buffers: config.AgmaAnalyticsBuffers{
			EventCount: cfg.Buffers.EventCount,
			BufferSize: cfg.Buffers.BufferSize,
			Timeout:    cfg.Buffers.Timeout,
		},
		Accounts: cfg.Accounts,
	}
	return base.NewModule(deps.HTTPClient, full, deps.Clock.(clock.Clock))
}
