package doohimpressionvalue

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const (
	defaultLookupPath              = "dooh.id"
	defaultOverwritePolicy         = overwritePolicyMissingOnly
	defaultTimeoutMS               = 100
	defaultCacheTTLSeconds         = 300
	defaultNegativeCacheTTLSeconds = 30
	defaultCacheSizeBytes          = 10 * 1024 * 1024
)

const (
	lookupPathDOOHID          = "dooh.id"
	lookupPathDOOHName        = "dooh.name"
	lookupPathDOOHPublisherID = "dooh.publisher.id"
	lookupPathImpID           = "imp.id"
	lookupPathImpTagID        = "imp.tagid"
)

type overwritePolicy string

const (
	overwritePolicyMissingOnly overwritePolicy = "missing_only"
	overwritePolicyAlways      overwritePolicy = "always"
)

type moduleConfig struct {
	Enabled                 bool              `json:"enabled,omitempty"`
	Endpoint                string            `json:"endpoint"`
	LookupPaths             []string          `json:"lookup_paths"`
	OverwritePolicy         overwritePolicy   `json:"overwrite_policy"`
	TimeoutMS               int               `json:"timeout_ms"`
	CacheTTLSeconds         int               `json:"cache_ttl_seconds"`
	NegativeCacheTTLSeconds int               `json:"negative_cache_ttl_seconds"`
	CacheSizeBytes          int               `json:"cache_size_bytes"`
	Headers                 map[string]string `json:"headers,omitempty"`
}

func parseModuleConfig(data json.RawMessage) (moduleConfig, error) {
	cfg := moduleConfig{
		LookupPaths:             []string{defaultLookupPath},
		OverwritePolicy:         defaultOverwritePolicy,
		TimeoutMS:               defaultTimeoutMS,
		CacheTTLSeconds:         defaultCacheTTLSeconds,
		NegativeCacheTTLSeconds: defaultNegativeCacheTTLSeconds,
		CacheSizeBytes:          defaultCacheSizeBytes,
	}

	if len(data) > 0 {
		if err := jsonutil.UnmarshalValid(data, &cfg); err != nil {
			return cfg, fmt.Errorf("failed to parse config: %s", err)
		}
	}

	if cfg.Endpoint == "" {
		return cfg, fmt.Errorf("endpoint is required")
	}

	endpointURL, err := url.ParseRequestURI(cfg.Endpoint)
	if err != nil {
		return cfg, fmt.Errorf("endpoint is invalid: %s", err)
	}
	if endpointURL.Scheme != "http" && endpointURL.Scheme != "https" {
		return cfg, fmt.Errorf("endpoint scheme must be http or https")
	}

	if len(cfg.LookupPaths) == 0 {
		cfg.LookupPaths = []string{defaultLookupPath}
	}

	lookupPaths, err := normalizeLookupPaths(cfg.LookupPaths)
	if err != nil {
		return cfg, err
	}
	cfg.LookupPaths = lookupPaths

	switch cfg.OverwritePolicy {
	case "":
		cfg.OverwritePolicy = defaultOverwritePolicy
	case overwritePolicyMissingOnly, overwritePolicyAlways:
	default:
		return cfg, fmt.Errorf("overwrite_policy must be %q or %q", overwritePolicyMissingOnly, overwritePolicyAlways)
	}

	if cfg.TimeoutMS < 0 {
		return cfg, fmt.Errorf("timeout_ms cannot be negative")
	}
	if cfg.TimeoutMS == 0 {
		cfg.TimeoutMS = defaultTimeoutMS
	}

	if cfg.CacheTTLSeconds < 0 {
		return cfg, fmt.Errorf("cache_ttl_seconds cannot be negative")
	}
	if cfg.CacheTTLSeconds == 0 {
		cfg.CacheTTLSeconds = defaultCacheTTLSeconds
	}

	if cfg.NegativeCacheTTLSeconds < 0 {
		return cfg, fmt.Errorf("negative_cache_ttl_seconds cannot be negative")
	}
	if cfg.NegativeCacheTTLSeconds == 0 {
		cfg.NegativeCacheTTLSeconds = defaultNegativeCacheTTLSeconds
	}

	if cfg.CacheSizeBytes < 0 {
		return cfg, fmt.Errorf("cache_size_bytes cannot be negative")
	}
	if cfg.CacheSizeBytes == 0 {
		cfg.CacheSizeBytes = defaultCacheSizeBytes
	}

	for name := range cfg.Headers {
		if name == "" {
			return cfg, fmt.Errorf("headers cannot contain an empty header name")
		}
	}

	return cfg, nil
}

func normalizeLookupPaths(paths []string) ([]string, error) {
	lookupPaths := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))

	for _, path := range paths {
		if !isSupportedLookupPath(path) {
			return nil, fmt.Errorf("lookup path %q is not supported", path)
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		lookupPaths = append(lookupPaths, path)
	}

	return lookupPaths, nil
}

func isSupportedLookupPath(path string) bool {
	switch path {
	case lookupPathDOOHID, lookupPathDOOHName, lookupPathDOOHPublisherID, lookupPathImpID, lookupPathImpTagID:
		return true
	default:
		return false
	}
}
