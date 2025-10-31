package trafficshaping

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	t.Run("empty_raw_config", func(t *testing.T) {
		cfg, err := parseConfig(json.RawMessage{})
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		// Should have default values
		assert.Equal(t, 30000, cfg.RefreshMs)
		assert.Equal(t, 1000, cfg.RequestTimeoutMs)
		assert.Equal(t, "pbs", cfg.SampleSalt)
		assert.False(t, cfg.PruneUserIds)
		assert.Equal(t, 300000, cfg.GeoCacheTTLMS)
	})

	t.Run("unmarshal_error", func(t *testing.T) {
		cfg, err := parseConfig(json.RawMessage(`{"invalid": json}`))
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to unmarshal config")
	})

	t.Run("valid_config", func(t *testing.T) {
		rawConfig := json.RawMessage(`{
			"enabled": true,
			"endpoint": "http://example.com/config.json",
			"refresh_ms": 60000,
			"request_timeout_ms": 2000,
			"sample_salt": "custom-salt",
			"prune_user_ids": true,
			"geo_cache_ttl_ms": 600000
		}`)
		cfg, err := parseConfig(rawConfig)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.True(t, cfg.Enabled)
		assert.Equal(t, "http://example.com/config.json", cfg.Endpoint)
		assert.Equal(t, 60000, cfg.RefreshMs)
		assert.Equal(t, 2000, cfg.RequestTimeoutMs)
		assert.Equal(t, "custom-salt", cfg.SampleSalt)
		assert.True(t, cfg.PruneUserIds)
		assert.Equal(t, 600000, cfg.GeoCacheTTLMS)
	})

	t.Run("valid_config_with_base_endpoint", func(t *testing.T) {
		rawConfig := json.RawMessage(`{
			"base_endpoint": "http://example.com/",
			"refresh_ms": 30000,
			"request_timeout_ms": 1000,
			"sample_salt": "pbs"
		}`)
		cfg, err := parseConfig(rawConfig)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "http://example.com/", cfg.BaseEndpoint)
	})

	t.Run("nil_raw_config", func(t *testing.T) {
		cfg, err := parseConfig(nil)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		// Should have default values
		assert.Equal(t, 30000, cfg.RefreshMs)
	})
}

