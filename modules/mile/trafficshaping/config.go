package trafficshaping

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// Config represents the module configuration
type Config struct {
	Enabled           bool     `json:"enabled"`
	Endpoint          string   `json:"endpoint"`      // Deprecated: use BaseEndpoint for dynamic URL construction
	BaseEndpoint      string   `json:"base_endpoint"` // Base URL for dynamic config fetching
	RefreshMs         int      `json:"refresh_ms"`
	RequestTimeoutMs  int      `json:"request_timeout_ms"`
	PruneUserIds      bool     `json:"prune_user_ids"`
	SampleSalt        string   `json:"sample_salt"`
	AllowedCountries  []string `json:"allowed_countries"`
	GeoLookupEndpoint string   `json:"geo_lookup_endpoint"` // HTTP endpoint option
	GeoDBPath         string   `json:"geo_db_path"`         // MaxMind database path option
	GeoCacheTTLMS     int      `json:"geo_cache_ttl_ms"`    // Only used for HTTP resolver

	// Whitelist endpoints for pre-filtering
	GeoWhitelistEndpoint      string `json:"geo_whitelist_endpoint"`      // URL to fetch geo whitelist
	PlatformWhitelistEndpoint string `json:"platform_whitelist_endpoint"` // URL to fetch platform whitelist
	WhitelistRefreshMs        int    `json:"whitelist_refresh_ms"`        // Whitelist refresh interval (default: 300000ms = 5 min)

	// Cached map for fast lookup (built once at parse time)
	allowedCountriesMap map[string]struct{}
}

// parseConfig parses and validates the module configuration
func parseConfig(rawConfig json.RawMessage) (*Config, error) {
	cfg := &Config{
		RefreshMs:          30000,
		RequestTimeoutMs:   1000,
		SampleSalt:         "pbs",
		PruneUserIds:       false,
		GeoCacheTTLMS:      300000,
		WhitelistRefreshMs: 300000, // 5 minutes default
	}

	if len(rawConfig) == 0 {
		return cfg, nil
	}

	if err := jsonutil.Unmarshal(rawConfig, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	// Build cached map for fast lookup
	cfg.buildAllowedCountriesMap()

	return cfg, nil
}

// validateConfig validates the module configuration
func validateConfig(cfg *Config) error {
	// Support both old 'endpoint' and new 'base_endpoint' for backward compatibility
	if cfg.Endpoint == "" && cfg.BaseEndpoint == "" {
		return errors.New("either endpoint or base_endpoint is required")
	}

	// If both are set, base_endpoint takes precedence
	if cfg.BaseEndpoint != "" {
		// Ensure base_endpoint ends with /
		if !strings.HasSuffix(cfg.BaseEndpoint, "/") {
			cfg.BaseEndpoint = cfg.BaseEndpoint + "/"
		}
	}

	if cfg.RefreshMs < 1000 {
		return errors.New("refresh_ms must be at least 1000ms")
	}

	if cfg.RequestTimeoutMs < 100 {
		return errors.New("request_timeout_ms must be at least 100ms")
	}

	if cfg.SampleSalt == "" {
		return errors.New("sample_salt cannot be empty")
	}

	if cfg.GeoCacheTTLMS < 1000 {
		return errors.New("geo_cache_ttl_ms must be at least 1000ms")
	}

	// Validate whitelist config if enabled
	if cfg.GeoWhitelistEndpoint != "" || cfg.PlatformWhitelistEndpoint != "" {
		if cfg.GeoWhitelistEndpoint == "" || cfg.PlatformWhitelistEndpoint == "" {
			return errors.New("both geo_whitelist_endpoint and platform_whitelist_endpoint must be configured together")
		}
		if cfg.WhitelistRefreshMs < 1000 {
			return errors.New("whitelist_refresh_ms must be at least 1000ms")
		}
	}

	return nil
}

// IsDynamicMode returns true if the module is configured for dynamic URL construction
func (c *Config) IsDynamicMode() bool {
	return c.BaseEndpoint != ""
}

// GetRefreshInterval returns the refresh interval as a duration
func (c *Config) GetRefreshInterval() time.Duration {
	return time.Duration(c.RefreshMs) * time.Millisecond
}

// GetRequestTimeout returns the request timeout as a duration
func (c *Config) GetRequestTimeout() time.Duration {
	return time.Duration(c.RequestTimeoutMs) * time.Millisecond
}

// buildAllowedCountriesMap builds the cached map from the slice
func (c *Config) buildAllowedCountriesMap() {
	if len(c.AllowedCountries) == 0 {
		c.allowedCountriesMap = nil
		return
	}

	c.allowedCountriesMap = make(map[string]struct{}, len(c.AllowedCountries))
	for _, country := range c.AllowedCountries {
		c.allowedCountriesMap[country] = struct{}{}
	}
}

// GetAllowedCountriesMap returns a map of allowed countries for fast lookup
func (c *Config) GetAllowedCountriesMap() map[string]struct{} {
	// Lazy initialization for configs created without parseConfig
	if c.allowedCountriesMap == nil && len(c.AllowedCountries) > 0 {
		c.buildAllowedCountriesMap()
	}
	return c.allowedCountriesMap
}

// GeoEnabled returns true if geo lookup fallback is configured
func (c *Config) GeoEnabled() bool {
	return c.GeoLookupEndpoint != "" || c.GeoDBPath != ""
}

// GetGeoCacheTTL returns the geo cache TTL as duration
func (c *Config) GetGeoCacheTTL() time.Duration {
	return time.Duration(c.GeoCacheTTLMS) * time.Millisecond
}

// WhitelistEnabled returns true if whitelist pre-filtering is configured
func (c *Config) WhitelistEnabled() bool {
	return c.GeoWhitelistEndpoint != "" && c.PlatformWhitelistEndpoint != ""
}

// GetWhitelistRefreshInterval returns the whitelist refresh interval as duration
func (c *Config) GetWhitelistRefreshInterval() time.Duration {
	return time.Duration(c.WhitelistRefreshMs) * time.Millisecond
}
