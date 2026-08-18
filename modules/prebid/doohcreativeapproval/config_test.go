package doohcreativeapproval

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
	assert.Equal(t, []string{defaultPlatformDOOH}, cfg.Platforms)
	assert.Equal(t, defaultTimeoutMS, cfg.TimeoutMS)
	assert.Equal(t, defaultCacheSizeBytes, cfg.CacheSizeBytes)
	assert.Equal(t, defaultMaxConcurrentLookups, cfg.MaxConcurrentLookups)
	assert.Equal(t, defaultApprovedTTLSeconds, cfg.ApprovedTTLSeconds)
	assert.Equal(t, defaultRejectedTTLSeconds, cfg.RejectedTTLSeconds)
	assert.Equal(t, defaultPendingTTLSeconds, cfg.PendingTTLSeconds)
}

func TestParseModuleConfigErrors(t *testing.T) {
	tests := []struct {
		name        string
		config      json.RawMessage
		expectedErr string
	}{
		{
			name:        "invalid-json",
			config:      json.RawMessage(`invalid`),
			expectedErr: "failed to parse config",
		},
		{
			name:        "invalid-endpoint",
			config:      json.RawMessage(`{"endpoint":"://bad"}`),
			expectedErr: "endpoint is invalid",
		},
		{
			name:        "invalid-endpoint-scheme",
			config:      json.RawMessage(`{"endpoint":"ftp://example.com"}`),
			expectedErr: "endpoint scheme must be http or https",
		},
		{
			name:        "invalid-platform",
			config:      json.RawMessage(`{"platforms":["site"]}`),
			expectedErr: `platforms must contain only "dooh"`,
		},
		{
			name:        "negative-timeout",
			config:      json.RawMessage(`{"timeout_ms":-1}`),
			expectedErr: "timeout_ms cannot be negative",
		},
		{
			name:        "negative-cache-size",
			config:      json.RawMessage(`{"cache_size_bytes":-1}`),
			expectedErr: "cache_size_bytes cannot be negative",
		},
		{
			name:        "cache-size-below-freecache-minimum",
			config:      json.RawMessage(`{"cache_size_bytes":1024}`),
			expectedErr: "cache_size_bytes must be at least 524288",
		},
		{
			name:        "negative-max-concurrent-lookups",
			config:      json.RawMessage(`{"max_concurrent_lookups":-1}`),
			expectedErr: "max_concurrent_lookups cannot be negative",
		},
		{
			name:        "negative-approved-ttl",
			config:      json.RawMessage(`{"approved_ttl_seconds":-1}`),
			expectedErr: "approved_ttl_seconds cannot be negative",
		},
		{
			name:        "negative-rejected-ttl",
			config:      json.RawMessage(`{"rejected_ttl_seconds":-1}`),
			expectedErr: "rejected_ttl_seconds cannot be negative",
		},
		{
			name:        "negative-pending-ttl",
			config:      json.RawMessage(`{"pending_ttl_seconds":-1}`),
			expectedErr: "pending_ttl_seconds cannot be negative",
		},
		{
			name:        "empty-header-name",
			config:      json.RawMessage(`{"headers":{"":"value"}}`),
			expectedErr: "headers cannot contain an empty header name",
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

func TestApplyAccountConfig(t *testing.T) {
	base := testModuleConfig()
	base.Endpoint = "http://host.example.com"
	base.Headers = map[string]string{"X-Host": "host"}
	base.TimeoutMS = 200
	base.ApprovedTTLSeconds = 300
	base.RejectedTTLSeconds = 400
	base.PendingTTLSeconds = 500
	base.CacheSizeBytes = 2 * minimumCacheSizeBytes
	base.MaxConcurrentLookups = 3
	base.ExemptBidders = []string{"hostbidder"}

	accountConfig := json.RawMessage(`{
		"enabled": false,
		"endpoint": "https://account.example.com/approval",
		"headers": {"X-Account": "account"},
		"timeout_ms": 10,
		"max_concurrent_lookups": 99,
		"approved_ttl_seconds": 11,
		"rejected_ttl_seconds": 12,
		"pending_ttl_seconds": 13,
		"exempt_bidders": ["AppNexus", " appnexus ", "", "Rubicon"]
	}`)

	cfg, err := applyAccountConfig(base, accountConfig)

	require.NoError(t, err)
	assert.False(t, cfg.Enabled)
	assert.Equal(t, "https://account.example.com/approval", cfg.Endpoint)
	assert.Equal(t, map[string]string{"X-Account": "account"}, cfg.Headers)
	assert.Equal(t, 10, cfg.TimeoutMS)
	assert.Equal(t, 11, cfg.ApprovedTTLSeconds)
	assert.Equal(t, 12, cfg.RejectedTTLSeconds)
	assert.Equal(t, 13, cfg.PendingTTLSeconds)
	assert.Equal(t, 2*minimumCacheSizeBytes, cfg.CacheSizeBytes)
	assert.Equal(t, 3, cfg.MaxConcurrentLookups)
	assert.Equal(t, []string{"appnexus", "rubicon"}, cfg.ExemptBidders)
}

func TestApplyAccountConfigInvalid(t *testing.T) {
	_, err := applyAccountConfig(testModuleConfig(), json.RawMessage(`{"platforms":["app"]}`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), `platforms must contain only "dooh"`)
}

func TestTTLForStatus(t *testing.T) {
	cfg := testModuleConfig()
	cfg.ApprovedTTLSeconds = 1
	cfg.RejectedTTLSeconds = 2
	cfg.PendingTTLSeconds = 3

	assert.Equal(t, 1, ttlForStatus(cfg, approvalStatusApproved))
	assert.Equal(t, 2, ttlForStatus(cfg, approvalStatusRejected))
	assert.Equal(t, 3, ttlForStatus(cfg, approvalStatusPending))
	assert.Equal(t, 3, ttlForStatus(cfg, "unknown"))
}
