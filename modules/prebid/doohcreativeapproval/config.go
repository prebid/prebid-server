package doohcreativeapproval

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const (
	defaultPlatformDOOH       = "dooh"
	defaultTimeoutMS          = 100
	defaultCacheSizeBytes     = 10 * 1024 * 1024
	defaultApprovedTTLSeconds = 3600
	defaultRejectedTTLSeconds = 300
	defaultPendingTTLSeconds  = 60
)

type moduleConfig struct {
	Enabled            bool              `json:"enabled,omitempty"`
	Platforms          []string          `json:"platforms,omitempty"`
	Endpoint           string            `json:"endpoint,omitempty"`
	Headers            map[string]string `json:"headers,omitempty"`
	TimeoutMS          int               `json:"timeout_ms,omitempty"`
	CacheSizeBytes     int               `json:"cache_size_bytes,omitempty"`
	ApprovedTTLSeconds int               `json:"approved_ttl_seconds,omitempty"`
	RejectedTTLSeconds int               `json:"rejected_ttl_seconds,omitempty"`
	PendingTTLSeconds  int               `json:"pending_ttl_seconds,omitempty"`
	ExemptBidders      []string          `json:"exempt_bidders,omitempty"`
}

type moduleConfigOverlay struct {
	Enabled            *bool             `json:"enabled,omitempty"`
	Platforms          []string          `json:"platforms,omitempty"`
	Endpoint           *string           `json:"endpoint,omitempty"`
	Headers            map[string]string `json:"headers,omitempty"`
	TimeoutMS          *int              `json:"timeout_ms,omitempty"`
	ApprovedTTLSeconds *int              `json:"approved_ttl_seconds,omitempty"`
	RejectedTTLSeconds *int              `json:"rejected_ttl_seconds,omitempty"`
	PendingTTLSeconds  *int              `json:"pending_ttl_seconds,omitempty"`
	ExemptBidders      []string          `json:"exempt_bidders,omitempty"`
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
		Enabled:            true,
		Platforms:          []string{defaultPlatformDOOH},
		TimeoutMS:          defaultTimeoutMS,
		CacheSizeBytes:     defaultCacheSizeBytes,
		ApprovedTTLSeconds: defaultApprovedTTLSeconds,
		RejectedTTLSeconds: defaultRejectedTTLSeconds,
		PendingTTLSeconds:  defaultPendingTTLSeconds,
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
		if overlay.Platforms != nil {
			base.Platforms = overlay.Platforms
		}
		if overlay.Endpoint != nil {
			base.Endpoint = *overlay.Endpoint
		}
		if overlay.Headers != nil {
			base.Headers = overlay.Headers
		}
		if overlay.TimeoutMS != nil {
			base.TimeoutMS = *overlay.TimeoutMS
		}
		if overlay.ApprovedTTLSeconds != nil {
			base.ApprovedTTLSeconds = *overlay.ApprovedTTLSeconds
		}
		if overlay.RejectedTTLSeconds != nil {
			base.RejectedTTLSeconds = *overlay.RejectedTTLSeconds
		}
		if overlay.PendingTTLSeconds != nil {
			base.PendingTTLSeconds = *overlay.PendingTTLSeconds
		}
		if overlay.ExemptBidders != nil {
			base.ExemptBidders = overlay.ExemptBidders
		}
	}

	return normalizeModuleConfig(base)
}

func normalizeModuleConfig(cfg moduleConfig) (moduleConfig, error) {
	platforms, err := normalizePlatforms(cfg.Platforms)
	if err != nil {
		return cfg, err
	}
	cfg.Platforms = platforms

	if cfg.Endpoint != "" {
		endpointURL, err := url.ParseRequestURI(cfg.Endpoint)
		if err != nil {
			return cfg, fmt.Errorf("endpoint is invalid: %s", err)
		}
		if endpointURL.Scheme != "http" && endpointURL.Scheme != "https" {
			return cfg, fmt.Errorf("endpoint scheme must be http or https")
		}
	}

	if cfg.TimeoutMS < 0 {
		return cfg, fmt.Errorf("timeout_ms cannot be negative")
	}
	if cfg.TimeoutMS == 0 {
		cfg.TimeoutMS = defaultTimeoutMS
	}

	if cfg.CacheSizeBytes < 0 {
		return cfg, fmt.Errorf("cache_size_bytes cannot be negative")
	}
	if cfg.CacheSizeBytes == 0 {
		cfg.CacheSizeBytes = defaultCacheSizeBytes
	}

	if cfg.ApprovedTTLSeconds < 0 {
		return cfg, fmt.Errorf("approved_ttl_seconds cannot be negative")
	}
	if cfg.ApprovedTTLSeconds == 0 {
		cfg.ApprovedTTLSeconds = defaultApprovedTTLSeconds
	}

	if cfg.RejectedTTLSeconds < 0 {
		return cfg, fmt.Errorf("rejected_ttl_seconds cannot be negative")
	}
	if cfg.RejectedTTLSeconds == 0 {
		cfg.RejectedTTLSeconds = defaultRejectedTTLSeconds
	}

	if cfg.PendingTTLSeconds < 0 {
		return cfg, fmt.Errorf("pending_ttl_seconds cannot be negative")
	}
	if cfg.PendingTTLSeconds == 0 {
		cfg.PendingTTLSeconds = defaultPendingTTLSeconds
	}

	for name := range cfg.Headers {
		if strings.TrimSpace(name) == "" {
			return cfg, fmt.Errorf("headers cannot contain an empty header name")
		}
	}

	cfg.ExemptBidders = normalizeExemptBidders(cfg.ExemptBidders)

	return cfg, nil
}

func normalizePlatforms(platforms []string) ([]string, error) {
	if len(platforms) == 0 {
		return []string{defaultPlatformDOOH}, nil
	}

	seen := make(map[string]struct{}, len(platforms))
	normalized := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		platform = strings.ToLower(strings.TrimSpace(platform))
		if platform == "" {
			return nil, fmt.Errorf("platforms cannot contain an empty platform")
		}
		if platform != defaultPlatformDOOH {
			return nil, fmt.Errorf("platforms must contain only %q", defaultPlatformDOOH)
		}
		if _, ok := seen[platform]; ok {
			continue
		}
		seen[platform] = struct{}{}
		normalized = append(normalized, platform)
	}

	return normalized, nil
}

func normalizeExemptBidders(bidders []string) []string {
	seen := make(map[string]struct{}, len(bidders))
	normalized := make([]string, 0, len(bidders))
	for _, bidder := range bidders {
		bidder = strings.ToLower(strings.TrimSpace(bidder))
		if bidder == "" {
			continue
		}
		if _, ok := seen[bidder]; ok {
			continue
		}
		seen[bidder] = struct{}{}
		normalized = append(normalized, bidder)
	}
	return normalized
}

func isBidderExempt(cfg moduleConfig, bidder string) bool {
	for _, exemptBidder := range cfg.ExemptBidders {
		if strings.EqualFold(exemptBidder, bidder) {
			return true
		}
	}
	return false
}

func ttlForStatus(cfg moduleConfig, status approvalStatus) int {
	switch status {
	case approvalStatusApproved:
		return cfg.ApprovedTTLSeconds
	case approvalStatusRejected:
		return cfg.RejectedTTLSeconds
	default:
		return cfg.PendingTTLSeconds
	}
}
