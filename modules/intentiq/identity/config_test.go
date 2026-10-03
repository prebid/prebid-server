package identity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/util/jsonutil"

	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider/aerospike"
)

func TestDefaultConfig(t *testing.T) {
	c := defaultConfig()
	assert.Equal(t, int64(1000), c.Timeout)
	assert.True(t, c.MetricsEnabled)
	assert.False(t, c.TraceEnabled)
	assert.Equal(t, 43_200, c.Cache.TTLSeconds)
	assert.Equal(t, 10, c.Cache.MaxKeys)
	assert.Equal(t, 100_000, c.Cache.MaxSize)
	assert.Equal(t, 86_400, c.Cache.TTLCeilingFirstPartySeconds)
	assert.Equal(t, 1_800, c.Cache.InProgressTTLSeconds)
	assert.False(t, c.Cache.Enabled)
	assert.Nil(t, c.Redis)
}

func TestConfigResolve(t *testing.T) {
	host := defaultConfig()
	host.APIEndpoint = "http://host"
	host.PartnerID = "host-partner"

	t.Run("nil account returns host config unchanged", func(t *testing.T) {
		assert.Equal(t, host, host.resolve(nil))
	})

	t.Run("empty account returns host config unchanged", func(t *testing.T) {
		assert.Equal(t, host, host.resolve([]byte{}))
	})

	t.Run("account overrides only present keys, merges nested cache", func(t *testing.T) {
		got := host.resolve([]byte(`{"partner_id":"acct","cache":{"ttl_seconds":60}}`))
		assert.Equal(t, "acct", got.PartnerID)          // overridden
		assert.Equal(t, "http://host", got.APIEndpoint) // retained
		assert.Equal(t, 60, got.Cache.TTLSeconds)       // nested override
		assert.Equal(t, 10, got.Cache.MaxKeys)          // nested default retained
		assert.Equal(t, int64(1000), got.Timeout)       // retained
	})

	t.Run("invalid JSON falls back to host config", func(t *testing.T) {
		assert.Equal(t, host, host.resolve([]byte(`{not json`)))
	})
}

func TestConfigTimeout(t *testing.T) {
	assert.Equal(t, 1500*time.Millisecond, Config{Timeout: 1500}.GetTimeout())
}

func TestConfigValkeyParsedFromJSON(t *testing.T) {
	c := defaultConfig()
	require.NoError(t, jsonutil.Unmarshal(
		[]byte(`{"cache":{"provider":"valkey"},"valkey":{"host":"vk1","port":6379,"password":"s3cr3t"}}`), &c))

	require.NotNil(t, c.Valkey)
	assert.Equal(t, "vk1", c.Valkey.Host)
	assert.Equal(t, 6379, c.Valkey.Port)
	assert.Equal(t, "s3cr3t", c.Valkey.Password)
	assert.Nil(t, c.Redis, "a valkey block must not imply a redis one")

	cfgs := c.GetCacheConfigs()
	assert.Equal(t, provider.CacheTypeValkey, c.GetCacheType())
	assert.Same(t, c.Valkey, cfgs.Valkey, "the selected block is passed through, not copied")
	assert.Nil(t, cfgs.Redis)
	assert.Nil(t, cfgs.Aerospike)
}

func TestConfigAerospikeParsedFromJSON(t *testing.T) {
	c := defaultConfig()
	require.NoError(t, jsonutil.Unmarshal(
		[]byte(`{"aerospike":{"host":"as1","port":3000,"namespace":"prebid","set":"iiq"}}`), &c))

	require.NotNil(t, c.Aerospike)
	assert.Equal(t, "as1", c.Aerospike.Host)
	assert.Equal(t, "iiq", c.Aerospike.Set)
	assert.Nil(t, c.Redis, "an aerospike block must not imply a redis one")
}

func TestConfigAerospikeClientPolicyMapping(t *testing.T) {
	c := defaultConfig()
	require.NoError(t, jsonutil.Unmarshal([]byte(`{"aerospike":{
		"host":"as1","port":3000,"namespace":"prebid","set":"iiq",
		"client_policy":{"connection_queue_size":256,"min_connections_per_node":4,
			"connect_timeout_ms":200,"idle_timeout_ms":5000}}}`), &c))

	p := c.Aerospike.Policy
	assert.Equal(t, 256, p.ConnectionQueueSize)
	assert.Equal(t, 4, p.MinConnectionsPerNode)
	assert.Equal(t, 200, p.ConnectTimeoutMs)
	assert.Equal(t, 5000, p.IdleTimeoutMs)
}

// An aerospike block with no client_policy must still be usable; the provider fills the defaults.
func TestConfigAerospikeWithoutClientPolicy(t *testing.T) {
	c := Config{Aerospike: &aerospike.Config{Host: "as1", Port: 3000, Namespace: "ns", Set: "iiq"}}

	require.NotNil(t, c.Aerospike)
	assert.Zero(t, c.Aerospike.Policy, "unset policy stays zero so the provider applies its defaults")
}
