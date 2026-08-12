package rtd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigDefaults(t *testing.T) {
	cfg, err := newConfig(json.RawMessage(`{"api_key": "key"}`))
	require.NoError(t, err)

	assert.Equal(t, DefaultEndpoint, cfg.Endpoint)
	assert.Equal(t, DefaultModel, cfg.Model)
	assert.Equal(t, DefaultDataProviderName, cfg.DataProviderName)
	assert.Equal(t, defaultTimeoutMs, cfg.Timeout)
	assert.Equal(t, defaultCacheTTLSeconds, cfg.CacheTTLSeconds)
	assert.Equal(t, defaultNegativeCacheTTLSeconds, cfg.NegativeCacheTTLSeconds)
	assert.Equal(t, defaultRetryCacheTTLSeconds, cfg.RetryCacheTTLSeconds)
	assert.Equal(t, defaultCacheSize, cfg.CacheSize)
	assert.Equal(t, defaultMinScore, cfg.MinScore)
	assert.Zero(t, cfg.MaxSegments)
	assert.False(t, cfg.EnrichContent10)
	assert.False(t, cfg.EnrichUserAudience)
}

func TestNewConfigOverrides(t *testing.T) {
	raw := json.RawMessage(`{
		"api_key": "key",
		"endpoint": "https://example.com/v1/responses",
		"model": "custom-model",
		"timeout_ms": 250,
		"cache_ttl_seconds": 10,
		"negative_cache_ttl_seconds": 11,
		"retry_cache_ttl_seconds": 12,
		"cache_size": 1048576,
		"min_score": 0.9,
		"max_segments": 3,
		"data_provider_name": "custom.example",
		"enrich_content_1_0": true,
		"enrich_user_audience": true,
		"account_filter": {"allow_list": ["1001"]}
	}`)

	cfg, err := newConfig(raw)
	require.NoError(t, err)

	assert.Equal(t, "https://example.com/v1/responses", cfg.Endpoint)
	assert.Equal(t, "custom-model", cfg.Model)
	assert.Equal(t, 250, cfg.Timeout)
	assert.Equal(t, 10, cfg.CacheTTLSeconds)
	assert.Equal(t, 11, cfg.NegativeCacheTTLSeconds)
	assert.Equal(t, 12, cfg.RetryCacheTTLSeconds)
	assert.Equal(t, 1048576, cfg.CacheSize)
	assert.InDelta(t, 0.9, cfg.MinScore, 0.0001)
	assert.Equal(t, 3, cfg.MaxSegments)
	assert.Equal(t, "custom.example", cfg.DataProviderName)
	assert.True(t, cfg.EnrichContent10)
	assert.True(t, cfg.EnrichUserAudience)
	assert.Equal(t, []string{"1001"}, cfg.AccountFilter.AllowList)
}

func TestNewConfigErrors(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{"malformed json", `{"api_key":`, "failed to parse config"},
		{"empty config", `{}`, "api_key is required"},
		{"missing api key", `{"endpoint":"https://example.com"}`, "api_key is required"},
		{"non http scheme", `{"api_key":"k","endpoint":"ftp://example.com/x"}`, "must be an http or https URL"},
		{"no host", `{"api_key":"k","endpoint":"https:///v1/responses"}`, "must include a host"},
		{"bad url", `{"api_key":"k","endpoint":"https://exa mple.com"}`, "not a valid URL"},
		{"negative timeout", `{"api_key":"k","timeout_ms":-1}`, "timeout_ms cannot be negative"},
		{"negative cache ttl", `{"api_key":"k","cache_ttl_seconds":-1}`, "cache_ttl_seconds cannot be negative"},
		{"negative negative ttl", `{"api_key":"k","negative_cache_ttl_seconds":-1}`, "negative_cache_ttl_seconds cannot be negative"},
		{"negative retry ttl", `{"api_key":"k","retry_cache_ttl_seconds":-1}`, "retry_cache_ttl_seconds cannot be negative"},
		{"cache too small", `{"api_key":"k","cache_size":1024}`, "cache_size must be at least"},
		{"min score too high", `{"api_key":"k","min_score":1.5}`, "min_score must be between 0 and 1"},
		{"min score negative", `{"api_key":"k","min_score":-0.5}`, "min_score must be between 0 and 1"},
		{"negative max segments", `{"api_key":"k","max_segments":-1}`, "max_segments cannot be negative"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := newConfig(json.RawMessage(test.raw))
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantErr)
		})
	}
}

func TestNewConfigEmptyRawMessage(t *testing.T) {
	_, err := newConfig(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_key is required")
}

func TestAccountFilterIsAllowed(t *testing.T) {
	tests := []struct {
		name      string
		allowList []string
		accountID string
		want      bool
	}{
		{"empty list allows all", nil, "1001", true},
		{"empty list allows unknown account", []string{}, "", true},
		{"listed account allowed", []string{"1001", "1002"}, "1002", true},
		{"unlisted account denied", []string{"1001"}, "9999", false},
		{"empty account denied when list set", []string{"1001"}, "", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filter := AccountFilter{AllowList: test.allowList}
			assert.Equal(t, test.want, filter.isAllowed(test.accountID))
		})
	}
}

func TestDefaultEndpointIsResponsesAPI(t *testing.T) {
	cfg, err := newConfig(json.RawMessage(`{"api_key": "key"}`))
	require.NoError(t, err)
	assert.Equal(t, "https://api.zerogpu.ai/v1/responses", cfg.Endpoint)
}
