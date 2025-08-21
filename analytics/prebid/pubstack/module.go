package pubstack

import (
	"encoding/json"

	"github.com/prebid/prebid-server/v3/analytics"
	base "github.com/prebid/prebid-server/v3/analytics/pubstack"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
)

// Config mirrors the existing top-level analytics.pubstack config subset needed to construct the module.
type Config struct {
	ScopeId     string `json:"scopeId"`
	IntakeUrl   string `json:"intakeUrl"`
	ConfRefresh string `json:"confRefresh"`
	Buffers     struct {
		EventCount int    `json:"eventCount"`
		BufferSize string `json:"bufferSize"`
		Timeout    string `json:"timeout"`
	} `json:"buffers"`
}

// Builder constructs the pubstack analytics module using provided config and dependencies.
func Builder(raw json.RawMessage, deps moduledeps.ModuleDeps) (analytics.Module, error) {
	if deps.HTTPClient == nil || deps.Clock == nil {
		return nil, nil // cannot build without required deps; silently skip
	}
	var cfg Config
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, err
		}
	}
	if cfg.IntakeUrl == "" || cfg.ScopeId == "" {
		return nil, nil
	}
	return base.NewModule(
		deps.HTTPClient,
		cfg.ScopeId,
		cfg.IntakeUrl,
		cfg.ConfRefresh,
		cfg.Buffers.EventCount,
		cfg.Buffers.BufferSize,
		cfg.Buffers.Timeout,
		deps.Clock,
	)
}
