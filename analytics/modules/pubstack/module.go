package pubstack

import (
	"encoding/json"

	"github.com/mitchellh/mapstructure"

	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/analytics/analyticsdeps"
)

type Config struct {
	Enabled     bool   `mapstructure:"enabled" json:"enabled"`
	ScopeId     string `mapstructure:"scopeId" json:"scopeId"`
	IntakeUrl   string `mapstructure:"intakeUrl" json:"intakeUrl"`
	ConfRefresh string `mapstructure:"confRefresh" json:"confRefresh"`
	Buffers     struct {
		EventCount int    `mapstructure:"eventCount" json:"eventCount"`
		BufferSize string `mapstructure:"bufferSize" json:"bufferSize"`
		Timeout    string `mapstructure:"timeout" json:"timeout"`
	} `mapstructure:"buffers" json:"buffers"`
}

func Builder(cfg json.RawMessage, deps analyticsdeps.Deps) (analytics.Module, error) {
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

	if c.IntakeUrl == "" || c.ScopeId == "" {
		return nil, nil
	}

	return NewModule(
		deps.HTTPClient,
		c.ScopeId,
		c.IntakeUrl,
		c.ConfRefresh,
		c.Buffers.EventCount,
		c.Buffers.BufferSize,
		c.Buffers.Timeout,
		deps.Clock,
	)
}
