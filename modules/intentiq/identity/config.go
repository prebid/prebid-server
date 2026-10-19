package identity

import (
	"encoding/json"
	"fmt"
	"time"

	iiqcache "github.com/Intent-IQ/identity-go/cache"
	iiqaerospike "github.com/Intent-IQ/identity-go/integrations/aerospike"
	iiqredis "github.com/Intent-IQ/identity-go/integrations/redis"
	iiqvalkey "github.com/Intent-IQ/identity-go/integrations/valkey"
)

type Config struct {
	PartnerID          string `json:"partner_id"`
	APIEndpoint        string `json:"api_endpoint"`
	ReportsEndpoint    string `json:"reports_endpoint"`
	TimeoutMS          int64  `json:"timeout"`
	WaitTimeoutMS      *int64 `json:"wait_timeout"`
	MaxConcurrentCalls int    `json:"max_concurrent_calls"`

	Cache     iiqcache.Config      `json:"cache"`
	Redis     *iiqredis.Config     `json:"redis"`
	Valkey    *iiqvalkey.Config    `json:"valkey"`
	Aerospike *iiqaerospike.Config `json:"aerospike"`

	MetricsEnabled bool            `json:"metrics_enabled"`
	TraceEnabled   bool            `json:"trace_enabled"`
	Reporting      ReportingConfig `json:"reporting"`
}

type ReportingConfig struct {
	MaxConcurrentCalls int   `json:"max_concurrent_calls"`
	ShutdownTimeoutMS  int64 `json:"shutdown_timeout"`
}

func defaultConfig() Config {
	return Config{
		TimeoutMS:      1000,
		MetricsEnabled: true,
		Reporting: ReportingConfig{
			MaxConcurrentCalls: 16,
			ShutdownTimeoutMS:  5000,
		},
		Cache: iiqcache.Config{
			TTLSeconds: 43_200, MaxKeys: 10, MaxSize: 100_000,
			TTLCeilingFirstPartySeconds: 86_400,
			TTLCeilingThirdPartySeconds: 43_200,
			TTLCeilingDeviceSeconds:     3_600,
			NegativeTTLSeconds:          120,
		},
	}
}

func (c Config) timeout() time.Duration { return time.Duration(c.TimeoutMS) * time.Millisecond }

func (c Config) waitTimeout() *time.Duration {
	if c.WaitTimeoutMS == nil {
		return nil
	}
	d := time.Duration(*c.WaitTimeoutMS) * time.Millisecond
	return &d
}

func (c Config) validate() error {
	if c.TimeoutMS <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if c.WaitTimeoutMS != nil && *c.WaitTimeoutMS < 0 {
		return fmt.Errorf("wait_timeout must not be negative")
	}
	if c.MaxConcurrentCalls < 0 {
		return fmt.Errorf("max_concurrent_calls must not be negative")
	}
	if c.Reporting.MaxConcurrentCalls <= 0 {
		return fmt.Errorf("reporting.max_concurrent_calls must be positive")
	}
	if c.Reporting.ShutdownTimeoutMS <= 0 {
		return fmt.Errorf("reporting.shutdown_timeout must be positive")
	}
	if c.WaitTimeoutMS != nil && *c.WaitTimeoutMS < c.TimeoutMS {
		if !c.Cache.Enabled {
			return fmt.Errorf("cache must be enabled when wait_timeout is less than timeout")
		}
		if c.MaxConcurrentCalls == 0 {
			return fmt.Errorf("max_concurrent_calls must be positive when wait_timeout is less than timeout")
		}
	}
	return nil
}

// resolve overlays account settings while keeping process-wide resource controls host-owned.
func (c Config) resolve(raw json.RawMessage) Config {
	if len(raw) == 0 {
		return c
	}
	resolved := c
	if err := json.Unmarshal(raw, &resolved); err != nil {
		return c
	}
	resolved.MaxConcurrentCalls = c.MaxConcurrentCalls
	resolved.Cache = c.Cache
	resolved.Redis, resolved.Valkey, resolved.Aerospike = c.Redis, c.Valkey, c.Aerospike
	resolved.MetricsEnabled = c.MetricsEnabled
	resolved.Reporting = c.Reporting
	if err := resolved.validate(); err != nil {
		return c
	}
	return resolved
}
