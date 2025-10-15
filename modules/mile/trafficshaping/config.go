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
	GeoLookupEndpoint string   `json:"geo_lookup_endpoint"`
	GeoCacheTTLMS     int      `json:"geo_cache_ttl_ms"`
}

// parseConfig parses and validates the module configuration
func parseConfig(rawConfig json.RawMessage) (*Config, error) {
	cfg := &Config{
		RefreshMs:        30000,
		RequestTimeoutMs: 1000,
		SampleSalt:       "pbs",
		PruneUserIds:     false,
		GeoCacheTTLMS:    300000,
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

// GetAllowedCountriesMap returns a map of allowed countries for fast lookup
func (c *Config) GetAllowedCountriesMap() map[string]struct{} {
	if len(c.AllowedCountries) == 0 {
		return nil
	}

	countries := make(map[string]struct{}, len(c.AllowedCountries))
	for _, country := range c.AllowedCountries {
		countries[country] = struct{}{}
	}
	return countries
}

// GeoEnabled returns true if geo lookup fallback is configured
func (c *Config) GeoEnabled() bool {
	return c.GeoLookupEndpoint != ""
}

// GetGeoCacheTTL returns the geo cache TTL as duration
func (c *Config) GetGeoCacheTTL() time.Duration {
	return time.Duration(c.GeoCacheTTLMS) * time.Millisecond
}
