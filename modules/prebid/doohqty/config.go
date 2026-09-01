package doohqty

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const (
	defaultLookupPath              = "dooh.id"
	defaultOverwritePolicy         = overwritePolicyMissingOnly
	defaultSourceType              = sourceTypeRequestLookup
	defaultTimeoutMS               = 100
	defaultCacheTTLSeconds         = 300
	defaultNegativeCacheTTLSeconds = 30
	defaultCacheSizeBytes          = 10 * 1024 * 1024
	defaultSyncRateSeconds         = 300
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

type sourceType string

const (
	sourceTypeRequestLookup sourceType = "request_lookup"
	sourceTypeCSVSnapshot   sourceType = "csv_snapshot"
)

type sourceConfig struct {
	Type            sourceType        `json:"type"`
	Endpoint        string            `json:"endpoint"`
	Headers         map[string]string `json:"headers,omitempty"`
	SyncRateSeconds int               `json:"sync_rate_seconds"`
}

type moduleConfig struct {
	Enabled                 bool            `json:"enabled,omitempty"`
	Source                  sourceConfig    `json:"source"`
	LookupPaths             []string        `json:"lookup_paths"`
	OverwritePolicy         overwritePolicy `json:"overwrite_policy"`
	TimeoutMS               int             `json:"timeout_ms"`
	CacheTTLSeconds         int             `json:"cache_ttl_seconds"`
	NegativeCacheTTLSeconds int             `json:"negative_cache_ttl_seconds"`
	CacheSizeBytes          int             `json:"cache_size_bytes"`
}

type moduleConfigOverlay struct {
	Enabled                 *bool                `json:"enabled,omitempty"`
	Source                  *sourceConfigOverlay `json:"source,omitempty"`
	LookupPaths             []string             `json:"lookup_paths,omitempty"`
	OverwritePolicy         *overwritePolicy     `json:"overwrite_policy,omitempty"`
	TimeoutMS               *int                 `json:"timeout_ms,omitempty"`
	CacheTTLSeconds         *int                 `json:"cache_ttl_seconds,omitempty"`
	NegativeCacheTTLSeconds *int                 `json:"negative_cache_ttl_seconds,omitempty"`
}

type sourceConfigOverlay struct {
	Type            *sourceType       `json:"type,omitempty"`
	Endpoint        *string           `json:"endpoint,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	SyncRateSeconds *int              `json:"sync_rate_seconds,omitempty"`
}

func parseModuleConfig(data json.RawMessage) (moduleConfig, error) {
	cfg := defaultModuleConfig()

	if len(data) > 0 {
		if err := jsonutil.UnmarshalValid(data, &cfg); err != nil {
			return cfg, fmt.Errorf("failed to parse config: %s", err)
		}
	}

	return normalizeModuleConfig(cfg)
}

func defaultModuleConfig() moduleConfig {
	return moduleConfig{
		Enabled: true,
		Source: sourceConfig{
			Type:            defaultSourceType,
			SyncRateSeconds: defaultSyncRateSeconds,
		},
		LookupPaths:             []string{defaultLookupPath},
		OverwritePolicy:         defaultOverwritePolicy,
		TimeoutMS:               defaultTimeoutMS,
		CacheTTLSeconds:         defaultCacheTTLSeconds,
		NegativeCacheTTLSeconds: defaultNegativeCacheTTLSeconds,
		CacheSizeBytes:          defaultCacheSizeBytes,
	}
}

func applyAccountConfig(base moduleConfig, data json.RawMessage) (moduleConfig, error) {
	if len(data) > 0 {
		var overlay moduleConfigOverlay
		if err := jsonutil.UnmarshalValid(data, &overlay); err != nil {
			return base, fmt.Errorf("failed to parse account config: %s", err)
		}

		if overlay.Enabled != nil {
			base.Enabled = *overlay.Enabled
		}
		if overlay.Source != nil {
			applySourceConfigOverlay(&base.Source, *overlay.Source)
		}
		if overlay.LookupPaths != nil {
			base.LookupPaths = overlay.LookupPaths
		}
		if overlay.OverwritePolicy != nil {
			base.OverwritePolicy = *overlay.OverwritePolicy
		}
		if overlay.TimeoutMS != nil {
			base.TimeoutMS = *overlay.TimeoutMS
		}
		if overlay.CacheTTLSeconds != nil {
			base.CacheTTLSeconds = *overlay.CacheTTLSeconds
		}
		if overlay.NegativeCacheTTLSeconds != nil {
			base.NegativeCacheTTLSeconds = *overlay.NegativeCacheTTLSeconds
		}
	}

	return normalizeModuleConfig(base)
}

func applySourceConfigOverlay(base *sourceConfig, overlay sourceConfigOverlay) {
	typeChanged := overlay.Type != nil && *overlay.Type != base.Type
	endpointChanged := overlay.Endpoint != nil && *overlay.Endpoint != base.Endpoint
	if typeChanged {
		base.Endpoint = ""
		base.Headers = nil
	}
	if endpointChanged && overlay.Headers == nil {
		base.Headers = nil
	}

	if overlay.Type != nil {
		base.Type = *overlay.Type
	}
	if overlay.Endpoint != nil {
		base.Endpoint = *overlay.Endpoint
	}
	if overlay.Headers != nil {
		base.Headers = overlay.Headers
	}
	if overlay.SyncRateSeconds != nil {
		base.SyncRateSeconds = *overlay.SyncRateSeconds
	}
}

func normalizeModuleConfig(cfg moduleConfig) (moduleConfig, error) {
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

	source, err := normalizeSourceConfig(cfg.Source)
	if err != nil {
		return cfg, err
	}
	cfg.Source = source

	return cfg, nil
}

func normalizeSourceConfig(cfg sourceConfig) (sourceConfig, error) {
	if cfg.Type == "" {
		cfg.Type = defaultSourceType
	}

	switch cfg.Type {
	case sourceTypeRequestLookup, sourceTypeCSVSnapshot:
	default:
		return cfg, fmt.Errorf("source.type must be %q or %q", sourceTypeRequestLookup, sourceTypeCSVSnapshot)
	}

	if cfg.Endpoint != "" {
		endpointURL, err := url.ParseRequestURI(cfg.Endpoint)
		if err != nil {
			return cfg, fmt.Errorf("source.endpoint is invalid: %s", err)
		}
		if endpointURL.Scheme != "http" && endpointURL.Scheme != "https" {
			return cfg, fmt.Errorf("source.endpoint scheme must be http or https")
		}
	}

	if cfg.SyncRateSeconds < 0 {
		return cfg, fmt.Errorf("source.sync_rate_seconds cannot be negative")
	}
	if cfg.SyncRateSeconds == 0 {
		cfg.SyncRateSeconds = defaultSyncRateSeconds
	}

	for name := range cfg.Headers {
		if name == "" {
			return cfg, fmt.Errorf("source.headers cannot contain an empty header name")
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
