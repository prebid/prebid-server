package identity

import (
	"time"

	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// Config is the effective module configuration. Host-level config is parsed in Builder (defaults
// applied there); per request the account-level config is merged over it by resolve. JSON keys are
// kebab-case to match the documented (Java) config surface so operators can reuse the same YAML.
type Config struct {
	APIEndpoint     string       `json:"api-endpoint"`
	ReportsEndpoint string       `json:"reports-endpoint"`
	PartnerID       string       `json:"partner-id"`
	Timeout         int64        `json:"timeout"` // ms
	Cache           CacheConfig  `json:"cache"`
	Redis           *RedisConfig `json:"redis"`
	CacheMaxSize    int          `json:"cache-max-size"` // in-process (L1) byte budget
	MetricsEnabled  bool         `json:"metrics-enabled"`
}

// CacheConfig configures the two-layer identity cache.
type CacheConfig struct {
	Enabled                     bool `json:"enabled"`
	TTLSeconds                  int  `json:"ttlseconds"`
	MaxKeys                     int  `json:"max-keys"`
	TTLCeilingFirstPartySeconds int  `json:"ttl-ceiling-first-party-seconds"`
	TTLCeilingThirdPartySeconds int  `json:"ttl-ceiling-third-party-seconds"`
	TTLCeilingDeviceSeconds     int  `json:"ttl-ceiling-device-seconds"`
	NegativeTTLSeconds          int  `json:"negative-ttl-seconds"`
	InProgressTTLSeconds        int  `json:"in-progress-ttl-seconds"`
}

// RedisConfig is the L2 backend connection. Host-level only (never overridden per account).
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
}

// defaultConfig returns a Config pre-populated with the module defaults (mirrors the Java property
// defaults). Builder unmarshals the host config over this, and resolve unmarshals account config
// over the resolved host config.
func defaultConfig() Config {
	return Config{
		Timeout:        1000,
		CacheMaxSize:   100_000,
		MetricsEnabled: true,
		Cache: CacheConfig{
			TTLSeconds:                  43_200,
			MaxKeys:                     10,
			TTLCeilingFirstPartySeconds: 86_400,
			TTLCeilingThirdPartySeconds: 43_200,
			TTLCeilingDeviceSeconds:     3_600,
			NegativeTTLSeconds:          120,
			InProgressTTLSeconds:        1_800,
		},
	}
}

// resolve merges the account-level module config over this (host-resolved) config and returns the
// effective config for the request. Absent keys retain the host value; nested objects merge
// field-by-field (Go unmarshal into an existing struct only overrides present keys). Account config
// never carries redis.* (host-level only), so the shared Redis pointer is not deep-copied.
func (c Config) resolve(accountConfig []byte) Config {
	if len(accountConfig) == 0 {
		return c
	}
	merged := c // value copy; nested structs copied by value
	if err := jsonutil.Unmarshal(accountConfig, &merged); err != nil {
		return c
	}
	return merged
}

func (c Config) timeout() time.Duration {
	return time.Duration(c.Timeout) * time.Millisecond
}

// ttlPolicy builds the cache TTL policy from the (seconds-based) cache config.
func (c Config) ttlPolicy() cache.TTLPolicy {
	sec := func(s int) time.Duration { return time.Duration(s) * time.Second }
	return cache.TTLPolicy{
		Default:           sec(c.Cache.TTLSeconds),
		FirstPartyCeiling: sec(c.Cache.TTLCeilingFirstPartySeconds),
		ThirdPartyCeiling: sec(c.Cache.TTLCeilingThirdPartySeconds),
		DeviceCeiling:     sec(c.Cache.TTLCeilingDeviceSeconds),
		NegativeTTL:       sec(c.Cache.NegativeTTLSeconds),
		InProgressTTL:     sec(c.Cache.InProgressTTLSeconds),
	}
}
