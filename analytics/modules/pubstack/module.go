package pubstack

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v3/util/jsonutil"

	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/analytics/analyticsdeps"
)

type Config struct {
	Enabled     bool   `json:"enabled"`
	ScopeId     string `json:"scopeId"`
	IntakeUrl   string `json:"intakeUrl"`
	ConfRefresh string `json:"confRefresh"`
	Buffers     struct {
		EventCount int    `json:"eventCount"`
		BufferSize string `json:"bufferSize"`
		Timeout    string `json:"timeout"`
	} `json:"buffers"`
}

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
