package rtd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"

	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const (
	// DefaultEndpoint is the ZeroGPU Responses API. It is overridable so a host
	// can point at a different region or a proxy, but the request and response
	// shapes are always those of the Responses API.
	DefaultEndpoint = "https://api.zerogpu.ai/v1/responses"

	// DefaultModel is the IAB domain classifier model.
	DefaultModel = "zlm-v1-iab-domain-classifier"

	// DefaultDataProviderName is written to the `name` field of every injected
	// ORTB data object so buyers can attribute the segments.
	DefaultDataProviderName = "zerogpu.ai"

	// defaultTimeoutMs bounds a background cache warm-up, not anything on the
	// auction path, so it is sized generously against measured API latency
	// rather than against the auction's latency budget.
	defaultTimeoutMs               = 2000
	defaultCacheTTLSeconds         = 86400 // 24h - domain classifications are stable
	defaultNegativeCacheTTLSeconds = 300   // empty result, 400/401/403/420
	defaultRetryCacheTTLSeconds    = 30    // timeout / 5xx - cold domains warm up server-side
	defaultCacheSize               = 10 * 1024 * 1024
	defaultMinScore                = 0.5

	// freecache rejects entries larger than 1/1024 of the cache size, so keep a
	// floor that comfortably holds a serialized classification.
	minCacheSize = 512 * 1024
)

// Config holds the host-level module configuration. Account-level config uses
// the same shape, but only AccountFilter and the enrichment toggles are
// meaningful there - credentials and cache sizing stay host-side.
type Config struct {
	APIKey   string `json:"api_key"`
	Endpoint string `json:"endpoint"`
	Model    string `json:"model"`

	Timeout                 int `json:"timeout_ms"`
	CacheTTLSeconds         int `json:"cache_ttl_seconds"`
	NegativeCacheTTLSeconds int `json:"negative_cache_ttl_seconds"`
	RetryCacheTTLSeconds    int `json:"retry_cache_ttl_seconds"`
	CacheSize               int `json:"cache_size"`

	MinScore         float64 `json:"min_score"`
	MaxSegments      int     `json:"max_segments"`
	DataProviderName string  `json:"data_provider_name"`

	// EnrichContent10 appends a second content data object carrying deprecated
	// IAB Content Taxonomy 1.0 codes under segtax 1.
	EnrichContent10 bool `json:"enrich_content_1_0"`

	// EnrichUserAudience appends IAB Audience Taxonomy 1.1 segments to
	// user.data under segtax 4. Off by default - see README for the privacy
	// rationale.
	EnrichUserAudience bool `json:"enrich_user_audience"`

	AccountFilter AccountFilter `json:"account_filter"`
}

// AccountFilter restricts a host-enabled module to a subset of accounts. An
// empty allow list means every account is served.
type AccountFilter struct {
	AllowList []string `json:"allow_list"`
}

// isAllowed reports whether the given account may use the module.
func (f AccountFilter) isAllowed(accountID string) bool {
	if len(f.AllowList) == 0 {
		return true
	}
	return slices.Contains(f.AllowList, accountID)
}

// newConfig unmarshals, defaults and validates the module configuration.
func newConfig(data json.RawMessage) (Config, error) {
	var cfg Config
	if len(data) > 0 {
		if err := jsonutil.UnmarshalValid(data, &cfg); err != nil {
			return cfg, fmt.Errorf("failed to parse config: %s", err)
		}
	}
	cfg.applyDefaults()
	return cfg, cfg.validate()
}

func (c *Config) applyDefaults() {
	if c.Endpoint == "" {
		c.Endpoint = DefaultEndpoint
	}
	if c.Model == "" {
		c.Model = DefaultModel
	}
	if c.DataProviderName == "" {
		c.DataProviderName = DefaultDataProviderName
	}
	if c.Timeout == 0 {
		c.Timeout = defaultTimeoutMs
	}
	if c.CacheTTLSeconds == 0 {
		c.CacheTTLSeconds = defaultCacheTTLSeconds
	}
	if c.NegativeCacheTTLSeconds == 0 {
		c.NegativeCacheTTLSeconds = defaultNegativeCacheTTLSeconds
	}
	if c.RetryCacheTTLSeconds == 0 {
		c.RetryCacheTTLSeconds = defaultRetryCacheTTLSeconds
	}
	if c.CacheSize == 0 {
		c.CacheSize = defaultCacheSize
	}
	if c.MinScore == 0 {
		c.MinScore = defaultMinScore
	}
}

func (c *Config) validate() error {
	if c.APIKey == "" {
		return errors.New("api_key is required")
	}

	parsed, err := url.Parse(c.Endpoint)
	if err != nil {
		return fmt.Errorf("endpoint is not a valid URL: %s", err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return errors.New("endpoint must be an http or https URL")
	}
	if parsed.Host == "" {
		return errors.New("endpoint must include a host")
	}

	if c.Timeout < 0 {
		return errors.New("timeout_ms cannot be negative")
	}
	if c.CacheTTLSeconds < 0 {
		return errors.New("cache_ttl_seconds cannot be negative")
	}
	if c.NegativeCacheTTLSeconds < 0 {
		return errors.New("negative_cache_ttl_seconds cannot be negative")
	}
	if c.RetryCacheTTLSeconds < 0 {
		return errors.New("retry_cache_ttl_seconds cannot be negative")
	}
	if c.CacheSize < minCacheSize {
		return fmt.Errorf("cache_size must be at least %d bytes", minCacheSize)
	}
	if c.MinScore < 0 || c.MinScore > 1 {
		return errors.New("min_score must be between 0 and 1")
	}
	if c.MaxSegments < 0 {
		return errors.New("max_segments cannot be negative")
	}
	return nil
}
