package pubstack

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v3/util/jsonutil"

	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/analytics/analyticsdeps"
)

type Config struct {
	Enabled     bool   `"enabled" json:"enabled"`
	ScopeId     string `"scopeId" json:"scopeId"`
	IntakeUrl   string `"intakeUrl" json:"intakeUrl"`
	ConfRefresh string `"confRefresh" json:"confRefresh"`
	Buffers     struct {
		EventCount int    `"eventCount" json:"eventCount"`
		BufferSize string `"bufferSize" json:"bufferSize"`
		Timeout    string `"timeout" json:"timeout"`
	} `"buffers" json:"buffers"`
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
