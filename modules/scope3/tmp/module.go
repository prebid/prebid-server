// Package tmp implements a Prebid Server module for AdCP Trusted Match Protocol.
package tmp

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coocood/freecache"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
)

const (
	defaultRouterURL     = "https://tmp.interchange.io"
	defaultTimeoutMs     = 200
	defaultCacheTTLSecs  = 60
	defaultCacheSize     = 10 * 1024 * 1024 // 10 MB
	maxIdentitiesPerSpec = 3
)

// Config holds module configuration.
type Config struct {
	RouterURL       string        `json:"router_url"`
	SellerAgentURL  string        `json:"seller_agent_url"`
	AuthKey         string        `json:"auth_key"`
	TimeoutMs       int           `json:"timeout_ms"`
	CacheTTLSeconds int           `json:"cache_ttl_seconds"`
	CacheSize       int           `json:"cache_size"`
	AddToTargeting  bool          `json:"add_to_targeting"`
	Masking         MaskingConfig `json:"masking"`
}

// MaskingConfig controls masking of user data before forwarding to the router.
type MaskingConfig struct {
	Enabled bool                `json:"enabled"`
	Geo     GeoMaskingConfig    `json:"geo"`
	User    UserMaskingConfig   `json:"user"`
	Device  DeviceMaskingConfig `json:"device"`
}

// GeoMaskingConfig controls geographic masking.
type GeoMaskingConfig struct {
	PreserveMetro    bool `json:"preserve_metro"`
	PreserveZip      bool `json:"preserve_zip"`
	PreserveCity     bool `json:"preserve_city"`
	LatLongPrecision int  `json:"lat_long_precision"`
}

// UserMaskingConfig controls user data masking.
type UserMaskingConfig struct {
	PreserveEids []string `json:"preserve_eids"`
}

// DeviceMaskingConfig controls device-identifier masking.
type DeviceMaskingConfig struct {
	PreserveMobileIds bool `json:"preserve_mobile_ids"`
}

// Module implements the Scope3 TMP module.
type Module struct {
	cfg        Config
	httpClient *http.Client
	cache      *freecache.Cache
	sha256Pool *sync.Pool
}

// Builder is the entry point for the module.
func Builder(rawCfg json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	var cfg Config
	if err := json.Unmarshal(rawCfg, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	if err := validate(&cfg); err != nil {
		return nil, err
	}
	defaults(&cfg)

	httpClient := &http.Client{
		Timeout: time.Duration(cfg.TimeoutMs) * time.Millisecond,
	}
	if deps.HTTPClient != nil && deps.HTTPClient.Transport != nil {
		httpClient.Transport = deps.HTTPClient.Transport
	}

	return &Module{
		cfg:        cfg,
		httpClient: httpClient,
		cache:      freecache.NewCache(cfg.CacheSize),
		sha256Pool: &sync.Pool{New: func() any { return sha256.New() }},
	}, nil
}

func validate(cfg *Config) error {
	if cfg.RouterURL == "" {
		return errors.New("router_url is required")
	}
	if cfg.SellerAgentURL == "" {
		return errors.New("seller_agent_url is required")
	}
	if cfg.TimeoutMs < 0 {
		return errors.New("timeout_ms must be positive")
	}
	if cfg.CacheSize < 0 {
		return errors.New("cache_size must be non-negative")
	}
	if cfg.Masking.Enabled {
		if cfg.Masking.Geo.LatLongPrecision < 0 {
			return errors.New("lat_long_precision cannot be negative")
		}
		if cfg.Masking.Geo.LatLongPrecision > 4 {
			return errors.New("lat_long_precision cannot exceed 4 decimal places for privacy protection")
		}
		if len(cfg.Masking.User.PreserveEids) > maxIdentitiesPerSpec {
			return fmt.Errorf("preserve_eids exceeds spec limit of %d entries", maxIdentitiesPerSpec)
		}
	}
	return nil
}

func defaults(cfg *Config) {
	if cfg.RouterURL == "" {
		cfg.RouterURL = defaultRouterURL
	}
	if cfg.TimeoutMs == 0 {
		cfg.TimeoutMs = defaultTimeoutMs
	}
	if cfg.CacheTTLSeconds == 0 {
		cfg.CacheTTLSeconds = defaultCacheTTLSecs
	}
	if cfg.CacheSize == 0 {
		cfg.CacheSize = defaultCacheSize
	}
	if cfg.Masking.Enabled {
		if cfg.Masking.Geo.LatLongPrecision == 0 {
			cfg.Masking.Geo.LatLongPrecision = 2
		}
		if len(cfg.Masking.User.PreserveEids) == 0 {
			cfg.Masking.User.PreserveEids = []string{"liveramp.com", "uidapi.com", "id5-sync.com"}
		}
	}
}
