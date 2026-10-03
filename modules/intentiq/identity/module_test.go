package identity

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/modules/moduledeps"
)

// registeredSeries returns the metric family names the module registered into reg.
func registeredSeries(t *testing.T, reg *prometheus.Registry) []string {
	t.Helper()
	families, err := reg.Gather()
	require.NoError(t, err)

	names := make([]string, 0, len(families))
	for _, family := range families {
		names = append(names, family.GetName())
	}
	return names
}

func TestBuilder_DefaultsNoCache(t *testing.T) {
	reg := prometheus.NewRegistry()
	raw := json.RawMessage(`{"api_endpoint":"http://x","partner_id":"p1"}`)
	built, err := Builder(raw, moduledeps.ModuleDeps{MetricsRegisterer: reg})
	require.NoError(t, err)

	m, ok := built.(*Module)
	require.True(t, ok)
	assert.Equal(t, "http://x", m.cfg.APIEndpoint)
	assert.Equal(t, int64(1000), m.cfg.Timeout) // default applied
	assert.Nil(t, m.cache, "cache disabled by default")

	m.metrics.Enriched("p1") // give the gather something to report
	assert.Contains(t, registeredSeries(t, reg), metrics.Prefix+"enriched_total",
		"metrics_enabled defaults true -> collectors land in the host registry")

	require.NoError(t, m.Shutdown()) // no store -> nil
}

func TestBuilder_InvalidConfig(t *testing.T) {
	_, err := Builder(json.RawMessage(`{bad`), moduledeps.ModuleDeps{})
	require.Error(t, err)
}

func TestBuilder_IgnoresFrameworkEnabledKey(t *testing.T) {
	raw := json.RawMessage(`{"enabled":true,"api_endpoint":"http://x","partner_id":"p1"}`)
	built, err := Builder(raw, moduledeps.ModuleDeps{})
	require.NoError(t, err)
	assert.Equal(t, "http://x", built.(*Module).cfg.APIEndpoint)
}

func TestBuilder_MetricsDisabledRegistersNothing(t *testing.T) {
	reg := prometheus.NewRegistry()
	built, err := Builder(json.RawMessage(`{"api_endpoint":"http://x","metrics_enabled":false}`),
		moduledeps.ModuleDeps{MetricsRegisterer: reg})
	require.NoError(t, err)

	m := built.(*Module)
	assert.NotPanics(t, func() { m.metrics.Enriched("p1") }, "recording must stay safe when disabled")
	assert.Empty(t, registeredSeries(t, reg), "nothing may be registered when metrics are disabled")
}

// A host that exposes no Prometheus registry leaves MetricsRegisterer nil. The module must still
// build and serve auctions, just without metrics.
func TestBuilder_NoRegistererStillBuilds(t *testing.T) {
	built, err := Builder(json.RawMessage(`{"api_endpoint":"http://x","partner_id":"p1"}`),
		moduledeps.ModuleDeps{})
	require.NoError(t, err)

	m := built.(*Module)
	assert.NotPanics(t, func() { m.metrics.Enriched("p1") }, "recording must stay safe with no registerer")
	require.NoError(t, m.Shutdown())
}

func TestBuilder_CacheEnabledWithRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	port, _ := strconv.Atoi(mr.Port())
	cfg := map[string]any{
		"api_endpoint": "http://x",
		"partner_id":   "p1",
		"cache":        map[string]any{"enabled": true, "provider": "redis"},
		"redis":        map[string]any{"host": mr.Host(), "port": port},
	}
	raw, _ := json.Marshal(cfg)

	built, err := Builder(raw, moduledeps.ModuleDeps{})
	require.NoError(t, err)

	m := built.(*Module)
	require.NotNil(t, m.cache, "cache should be built when enabled + redis configured")
	require.NotNil(t, m.store)
	assert.Equal(t, "redis", m.store.Name())

	require.NoError(t, m.Shutdown()) // closes the store
}

func TestBuilder_CacheEnabledWithValkey(t *testing.T) {
	mr := miniredis.RunT(t)
	port, _ := strconv.Atoi(mr.Port())
	cfg := map[string]any{
		"api_endpoint": "http://x",
		"partner_id":   "p1",
		"cache":        map[string]any{"enabled": true, "provider": "valkey"},
		"valkey":       map[string]any{"host": mr.Host(), "port": port},
	}
	raw, _ := json.Marshal(cfg)

	built, err := Builder(raw, moduledeps.ModuleDeps{})
	require.NoError(t, err)

	m := built.(*Module)
	require.NotNil(t, m.cache, "cache should be built when enabled + valkey configured")
	require.NotNil(t, m.store)
	assert.Equal(t, "valkey", m.store.Name())

	require.NoError(t, m.Shutdown())
}

// Enabling the cache without naming a provider is a misconfiguration: fail startup rather than
// silently running L1-only.
func TestBuilder_CacheEnabledWithoutProviderFailsFast(t *testing.T) {
	raw := json.RawMessage(`{"api_endpoint":"http://x","partner_id":"p1","cache":{"enabled":true}}`)

	_, err := Builder(raw, moduledeps.ModuleDeps{})
	require.ErrorContains(t, err, "unknown cache type")
}

func TestBuilder_CacheEnabledUnreachableBackendFailsFast(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"api_endpoint": "http://x",
		"partner_id":   "p1",
		"cache":        map[string]any{"enabled": true, "provider": "redis"},
		"redis":        map[string]any{"host": "127.0.0.1", "port": 1},
	})

	_, err := Builder(raw, moduledeps.ModuleDeps{})
	require.Error(t, err)
}

// With cache.provider set, every block may sit in one config file and only the named one is built.
func TestBuilder_CacheProviderSelectsBackend(t *testing.T) {
	mr := miniredis.RunT(t)
	port, _ := strconv.Atoi(mr.Port())
	raw, _ := json.Marshal(map[string]any{
		"api_endpoint": "http://x",
		"partner_id":   "p1",
		"cache":        map[string]any{"enabled": true, "provider": "redis"},
		"redis":        map[string]any{"host": mr.Host(), "port": port},
		"valkey":       map[string]any{"host": "vk1", "port": 6379},
		"aerospike":    map[string]any{"host": "as1", "port": 3000, "namespace": "prebid", "set": "iiq"},
	})

	built, err := Builder(raw, moduledeps.ModuleDeps{})
	require.NoError(t, err, "the selector must disambiguate instead of failing")

	m := built.(*Module)
	require.NotNil(t, m.store)
	assert.Equal(t, "redis", m.store.Name())
	require.NoError(t, m.Shutdown())
}

func TestBuilder_UnknownCacheProviderFailsFast(t *testing.T) {
	raw := json.RawMessage(`{"api_endpoint":"http://x","partner_id":"p1",
		"cache":{"enabled":true,"provider":"memcached"},"redis":{"host":"localhost","port":6379}}`)

	_, err := Builder(raw, moduledeps.ModuleDeps{})
	require.ErrorContains(t, err, "unknown cache type")
}
