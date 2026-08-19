package identity

import (
	"time"

	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider/aerospike"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider/redis"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider/valkey"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type Config struct {
	PartnerID string `json:"partner_id"`

	APIEndpoint     string `json:"api_endpoint"`
	ReportsEndpoint string `json:"reports_endpoint"`
	Timeout         int64  `json:"timeout"` // ms

	Cache     cache.Config      `json:"cache"`
	Redis     *redis.Config     `json:"redis"`
	Valkey    *valkey.Config    `json:"valkey"`
	Aerospike *aerospike.Config `json:"aerospike"`

	MetricsEnabled bool `json:"metrics_enabled"`

	TraceEnabled bool `json:"trace_enabled"`
}

func (c Config) GetCacheType() provider.CacheType {
	return provider.CacheType(c.Cache.Provider)
}

func (c Config) GetCacheConfigs() provider.Configs {
	return provider.Configs{Redis: c.Redis, Valkey: c.Valkey, Aerospike: c.Aerospike}
}

func (c Config) GetTimeout() time.Duration {
	return time.Duration(c.Timeout) * time.Millisecond
}

// defaultConfig returns a Config pre-populated with the module defaults (mirrors the Java property
// defaults). Builder unmarshals the host config over this, and resolve unmarshals account config
// over the resolved host config.
func defaultConfig() Config {
	return Config{
		Timeout:        1000,
		MetricsEnabled: true,
		TraceEnabled:   false,
		Cache: cache.Config{
			TTLSeconds:                  43_200,
			MaxKeys:                     10,
			MaxSize:                     100_000,
			TTLCeilingFirstPartySeconds: 86_400,
			TTLCeilingThirdPartySeconds: 43_200,
			TTLCeilingDeviceSeconds:     3_600,
			NegativeTTLSeconds:          120,
			InProgressTTLSeconds:        1_800,
		},
	}
}

// resolve merges the account-level module config over this (host-resolved) config
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
