package doohqty

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseModuleConfigDefaults(t *testing.T) {
	cfg, err := parseModuleConfig(nil)

	require.NoError(t, err)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, sourceTypeRequestLookup, cfg.Source.Type)
	assert.Equal(t, defaultSyncRateSeconds, cfg.Source.SyncRateSeconds)
	assert.Equal(t, []string{lookupPathDOOHID}, cfg.LookupPaths)
	assert.Equal(t, overwritePolicyMissingOnly, cfg.OverwritePolicy)
	assert.Equal(t, defaultTimeoutMS, cfg.TimeoutMS)
	assert.Equal(t, defaultCacheTTLSeconds, cfg.CacheTTLSeconds)
	assert.Equal(t, defaultNegativeCacheTTLSeconds, cfg.NegativeCacheTTLSeconds)
	assert.Equal(t, defaultCacheSizeBytes, cfg.CacheSizeBytes)
}

func TestParseModuleConfigValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		config      json.RawMessage
		expectedErr string
	}{
		{
			name:        "invalid-json",
			config:      json.RawMessage(`not-json`),
			expectedErr: "failed to parse config",
		},
		{
			name:        "unsupported-source-type",
			config:      json.RawMessage(`{"source":{"type":"static"}}`),
			expectedErr: `source.type must be "request_lookup" or "csv_snapshot"`,
		},
		{
			name:        "invalid-endpoint",
			config:      json.RawMessage(`{"source":{"endpoint":"://bad"}}`),
			expectedErr: "source.endpoint is invalid",
		},
		{
			name:        "invalid-endpoint-scheme",
			config:      json.RawMessage(`{"source":{"endpoint":"ftp://example.com"}}`),
			expectedErr: "source.endpoint scheme must be http or https",
		},
		{
			name:        "unsupported-lookup-path",
			config:      json.RawMessage(`{"lookup_paths":["site.id"]}`),
			expectedErr: `lookup path "site.id" is not supported`,
		},
		{
			name:        "unsupported-overwrite-policy",
			config:      json.RawMessage(`{"overwrite_policy":"replace"}`),
			expectedErr: `overwrite_policy must be "missing_only" or "always"`,
		},
		{
			name:        "negative-timeout",
			config:      json.RawMessage(`{"timeout_ms":-1}`),
			expectedErr: "timeout_ms cannot be negative",
		},
		{
			name:        "negative-cache-ttl",
			config:      json.RawMessage(`{"cache_ttl_seconds":-1}`),
			expectedErr: "cache_ttl_seconds cannot be negative",
		},
		{
			name:        "negative-negative-cache-ttl",
			config:      json.RawMessage(`{"negative_cache_ttl_seconds":-1}`),
			expectedErr: "negative_cache_ttl_seconds cannot be negative",
		},
		{
			name:        "negative-cache-size",
			config:      json.RawMessage(`{"cache_size_bytes":-1}`),
			expectedErr: "cache_size_bytes cannot be negative",
		},
		{
			name:        "negative-sync-rate",
			config:      json.RawMessage(`{"source":{"sync_rate_seconds":-1}}`),
			expectedErr: "source.sync_rate_seconds cannot be negative",
		},
		{
			name:        "empty-header-name",
			config:      json.RawMessage(`{"source":{"headers":{"":"token"}}}`),
			expectedErr: "source.headers cannot contain an empty header name",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseModuleConfig(test.config)

			require.Error(t, err)
			assert.Contains(t, err.Error(), test.expectedErr)
		})
	}
}

func TestApplyAccountConfigOverlay(t *testing.T) {
	base := defaultModuleConfig()
	base.Source.Endpoint = "https://host.example.com/lookup"
	base.Source.Headers = map[string]string{"X-Host": "host"}
	base.CacheSizeBytes = 12345

	cfg, err := applyAccountConfig(base, json.RawMessage(`{
		"enabled": false,
		"source": {
			"endpoint": "https://account.example.com/lookup",
			"headers": {"X-Account": "account"},
			"sync_rate_seconds": 60
		},
		"lookup_paths": ["imp.tagid", "dooh.id"],
		"overwrite_policy": "always",
		"timeout_ms": 25,
		"cache_ttl_seconds": 26,
		"negative_cache_ttl_seconds": 27
	}`))

	require.NoError(t, err)
	assert.False(t, cfg.Enabled)
	assert.Equal(t, sourceTypeRequestLookup, cfg.Source.Type)
	assert.Equal(t, "https://account.example.com/lookup", cfg.Source.Endpoint)
	assert.Equal(t, map[string]string{"X-Account": "account"}, cfg.Source.Headers)
	assert.Equal(t, 60, cfg.Source.SyncRateSeconds)
	assert.Equal(t, []string{lookupPathImpTagID, lookupPathDOOHID}, cfg.LookupPaths)
	assert.Equal(t, overwritePolicyAlways, cfg.OverwritePolicy)
	assert.Equal(t, 25, cfg.TimeoutMS)
	assert.Equal(t, 26, cfg.CacheTTLSeconds)
	assert.Equal(t, 27, cfg.NegativeCacheTTLSeconds)
	assert.Equal(t, 12345, cfg.CacheSizeBytes)
}

func TestApplyAccountConfigClearsInheritedSourceHeaders(t *testing.T) {
	base := defaultModuleConfig()
	base.Source.Type = sourceTypeRequestLookup
	base.Source.Endpoint = "https://host.example.com/lookup"
	base.Source.Headers = map[string]string{"Authorization": "Bearer host"}

	cfg, err := applyAccountConfig(base, json.RawMessage(`{
		"source": {"endpoint": "https://account.example.com/lookup"}
	}`))

	require.NoError(t, err)
	assert.Equal(t, "https://account.example.com/lookup", cfg.Source.Endpoint)
	assert.Nil(t, cfg.Source.Headers)

	cfg, err = applyAccountConfig(base, json.RawMessage(`{
		"source": {"type": "csv_snapshot"}
	}`))

	require.NoError(t, err)
	assert.Equal(t, sourceTypeCSVSnapshot, cfg.Source.Type)
	assert.Empty(t, cfg.Source.Endpoint)
	assert.Nil(t, cfg.Source.Headers)
}
