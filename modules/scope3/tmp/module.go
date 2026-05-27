// Package tmp implements a Prebid Server module for AdCP Trusted Match Protocol.
package tmp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/coocood/freecache"
	"github.com/prebid/prebid-server/v4/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/logger"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/util/iterutil"
	"github.com/tidwall/sjson"
)

const (
	defaultTimeoutMs       = 200
	defaultCacheTTLSecs    = 60
	defaultCacheSize       = 10 * 1024 * 1024 // 10 MB
	maxIdentitiesPerSpec   = 3
	moduleContextAsyncKey  = "scope3.tmp.AsyncRequest"
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
	if err := validateRouterURL(cfg.RouterURL); err != nil {
		return err
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

// Interface assertions.
var (
	_ hookstage.Entrypoint              = (*Module)(nil)
	_ hookstage.ProcessedAuctionRequest = (*Module)(nil)
	_ hookstage.AuctionResponse         = (*Module)(nil)
)

// HandleEntrypointHook initializes per-auction state.
func (m *Module) HandleEntrypointHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	mc := hookstage.NewModuleContext()
	ar := newAsyncRequest(payload.Request.Context())
	ar.module = m
	mc.Set(moduleContextAsyncKey, ar)
	return hookstage.HookResult[hookstage.EntrypointPayload]{ModuleContext: mc}, nil
}

// HandleProcessedAuctionHook starts the asynchronous TMP fan-out. Returns
// immediately while the goroutine runs in parallel with the bidder auction.
func (m *Module) HandleProcessedAuctionHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	var ret hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]

	stored, ok := miCtx.ModuleContext.Get(moduleContextAsyncKey)
	if !ok {
		return ret, nil
	}
	ar, ok := stored.(*AsyncRequest)
	if !ok {
		return ret, nil
	}

	requestExt := json.RawMessage(nil)
	if payload.Request != nil && payload.Request.BidRequest != nil {
		requestExt = payload.Request.BidRequest.Ext
	}

	ar.fetchAsync(payload.Request.BidRequest, miCtx.AccountConfig, requestExt)
	return ret, nil
}

// HandleAuctionResponseHook waits for the async fan-out to complete (bounded
// by the hook's context) and writes per-imp enrichment into the response.
func (m *Module) HandleAuctionResponseHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.AuctionResponsePayload,
) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	var ret hookstage.HookResult[hookstage.AuctionResponsePayload]

	stored, ok := miCtx.ModuleContext.Get(moduleContextAsyncKey)
	if !ok {
		return ret, nil
	}
	ar, ok := stored.(*AsyncRequest)
	if !ok {
		return ret, nil
	}
	defer ar.cancel()

	if ar.done == nil {
		// Processed-auction hook never ran (e.g., test bypass). Nothing to write.
		return ret, nil
	}

	select {
	case <-ar.done:
	case <-ctx.Done():
		logger.Warnf("scope3.tmp: auction context cancelled while waiting for TMP result (auction %s)", payload.BidResponse.ID)
		ret.AnalyticsTags = analyticsErrorTag("scope3_tmp_timeout", "auction context cancelled")
		return ret, nil
	}

	if ar.err != nil {
		logger.Warnf("scope3.tmp: no enrichment for auction %s due to error: %v", payload.BidResponse.ID, ar.err)
		ret.AnalyticsTags = analyticsErrorTag("scope3_tmp_fetch", ar.err.Error())
		return ret, nil
	}
	if ar.result == nil {
		return ret, nil
	}

	result := ar.result
	addToTargeting := m.cfg.AddToTargeting

	ret.ChangeSet.AddMutation(
		func(p hookstage.AuctionResponsePayload) (hookstage.AuctionResponsePayload, error) {
			if p.BidResponse.Ext == nil {
				p.BidResponse.Ext = []byte("{}")
			}
			if result.TMPX != "" {
				p.BidResponse.Ext, _ = sjson.SetBytes(p.BidResponse.Ext, "scope3.tmp.tmpx", result.TMPX)
			}

			for seatBid := range iterutil.SlicePointerValues(p.BidResponse.SeatBid) {
				for bid := range iterutil.SlicePointerValues(seatBid.Bid) {
					placement, ok := result.ImpToPlacement[bid.ImpID]
					if !ok {
						continue
					}
					pkg := result.PerPlacement[placement]
					if bid.Ext == nil {
						bid.Ext = []byte("{}")
					}
					bid.Ext, _ = sjson.SetBytes(bid.Ext, "scope3.tmp.placement_id", placement)
					bid.Ext, _ = sjson.SetBytes(bid.Ext, "scope3.tmp.eligible_packages", pkg.EligiblePackages)
					if len(pkg.Segments) > 0 {
						bid.Ext, _ = sjson.SetBytes(bid.Ext, "scope3.tmp.segments", pkg.Segments)
					}
					if addToTargeting {
						if result.TMPX != "" {
							bid.Ext, _ = sjson.SetBytes(bid.Ext, "prebid.targeting.TMPX", result.TMPX)
						}
						// Group targeting_kvs by key — repeated keys become arrays.
						grouped := map[string][]string{}
						order := []string{}
						for _, kv := range pkg.TargetingKVs {
							if _, seen := grouped[kv.Key]; !seen {
								order = append(order, kv.Key)
							}
							grouped[kv.Key] = append(grouped[kv.Key], kv.Value)
						}
						for _, key := range order {
							values := grouped[key]
							if len(values) == 1 {
								bid.Ext, _ = sjson.SetBytes(bid.Ext, "prebid.targeting."+key, values[0])
							} else {
								bid.Ext, _ = sjson.SetBytes(bid.Ext, "prebid.targeting."+key, values)
							}
						}
					}
				}
			}
			return p, nil
		},
		hookstage.MutationUpdate,
		"ext",
	)
	return ret, nil
}

// validateRouterURL requires the URL parse correctly and use HTTPS. Allows
// http for loopback hosts (localhost, 127.0.0.1, ::1) so unit tests using
// httptest.NewServer continue to work.
func validateRouterURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("router_url is not a valid URL: %w", err)
	}
	if u.Host == "" {
		return errors.New("router_url is missing a host")
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" && isLoopbackHost(u.Hostname()) {
		return nil
	}
	return fmt.Errorf("router_url must use https (got scheme %q)", u.Scheme)
}

func isLoopbackHost(host string) bool {
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

func analyticsErrorTag(name, msg string) hookanalytics.Analytics {
	return hookanalytics.Analytics{
		Activities: []hookanalytics.Activity{{
			Name:   name,
			Status: hookanalytics.ActivityStatusError,
			Results: []hookanalytics.Result{{
				Status: hookanalytics.ResultStatusError,
				Values: map[string]interface{}{"error": msg},
			}},
		}},
	}
}
