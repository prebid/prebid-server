package identity

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/modules/moduledeps"
)

func TestBuilder_DefaultsNoCache(t *testing.T) {
	raw := json.RawMessage(`{"api-endpoint":"http://x","partner-id":"p1"}`)
	built, err := Builder(raw, moduledeps.ModuleDeps{})
	require.NoError(t, err)

	m, ok := built.(*Module)
	require.True(t, ok)
	assert.Equal(t, "http://x", m.cfg.APIEndpoint)
	assert.Equal(t, int64(1000), m.cfg.Timeout) // default applied
	assert.Nil(t, m.cache, "cache disabled by default")
	assert.Nil(t, m.reporter)
	assert.IsType(t, noopMetrics{}, m.metrics, "nil registerer -> noop metrics")

	require.NoError(t, m.Shutdown()) // no reporter/redis -> nil
}

func TestBuilder_InvalidConfig(t *testing.T) {
	_, err := Builder(json.RawMessage(`{bad`), moduledeps.ModuleDeps{})
	require.Error(t, err)
}

func TestBuilder_MetricsRegistererYieldsPromMetrics(t *testing.T) {
	built, err := Builder(json.RawMessage(`{"api-endpoint":"http://x"}`),
		moduledeps.ModuleDeps{MetricsRegisterer: prometheus.NewRegistry()})
	require.NoError(t, err)
	assert.IsType(t, &promMetrics{}, built.(*Module).metrics)
}

func TestBuilder_CacheEnabledWithRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	port, _ := strconv.Atoi(mr.Port())
	cfg := map[string]any{
		"api-endpoint": "http://x",
		"partner-id":   "p1",
		"cache":        map[string]any{"enabled": true},
		"redis":        map[string]any{"host": mr.Host(), "port": port},
	}
	raw, _ := json.Marshal(cfg)

	built, err := Builder(raw, moduledeps.ModuleDeps{MetricsRegisterer: prometheus.NewRegistry()})
	require.NoError(t, err)

	m := built.(*Module)
	require.NotNil(t, m.cache, "cache should be built when enabled + redis configured")
	require.NotNil(t, m.reporter)
	require.NotNil(t, m.redisClient)

	require.NoError(t, m.Shutdown()) // stops reporter + closes redis client
}
