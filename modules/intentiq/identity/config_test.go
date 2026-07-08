package identity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	c := defaultConfig()
	assert.Equal(t, int64(1000), c.Timeout)
	assert.Equal(t, 100_000, c.CacheMaxSize)
	assert.True(t, c.MetricsEnabled)
	assert.Equal(t, 43_200, c.Cache.TTLSeconds)
	assert.Equal(t, 10, c.Cache.MaxKeys)
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
		got := host.resolve([]byte(`{"partner-id":"acct","cache":{"ttlseconds":60}}`))
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

func TestConfigTTLPolicy(t *testing.T) {
	c := defaultConfig()
	c.Cache.TTLSeconds = 100
	c.Cache.TTLCeilingFirstPartySeconds = 200
	c.Cache.TTLCeilingThirdPartySeconds = 300
	c.Cache.TTLCeilingDeviceSeconds = 400
	c.Cache.NegativeTTLSeconds = 5
	c.Cache.InProgressTTLSeconds = 6

	p := c.ttlPolicy()
	assert.Equal(t, 100*time.Second, p.Default)
	assert.Equal(t, 200*time.Second, p.FirstPartyCeiling)
	assert.Equal(t, 300*time.Second, p.ThirdPartyCeiling)
	assert.Equal(t, 400*time.Second, p.DeviceCeiling)
	assert.Equal(t, 5*time.Second, p.NegativeTTL)
	assert.Equal(t, 6*time.Second, p.InProgressTTL)
}

func TestConfigTimeout(t *testing.T) {
	assert.Equal(t, 1500*time.Millisecond, Config{Timeout: 1500}.timeout())
}
