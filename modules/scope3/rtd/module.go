// Package scope3 implements a Prebid Server module for Scope3 RTD
package scope3

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// Builder is the entry point for the module
// This is called by Prebid Server to initialize the module
func Builder(config json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	var cfg Config
	if err := jsonutil.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://rtdp.scope3.com/prebid/rtii"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 1000 // 1000ms default
	}
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 60 // 60 seconds default
	}

	// Set masking defaults
	if cfg.Masking.Geo.LatLongPrecision == 0 && cfg.Masking.Enabled {
		cfg.Masking.Geo.LatLongPrecision = 2 // 2 decimal places default (~1.1km precision)
	}
	if cfg.Masking.Enabled && len(cfg.Masking.User.PreserveEids) == 0 {
		// Default to preserving common identity providers
		cfg.Masking.User.PreserveEids = []string{"liveramp.com", "uidapi.com", "id5-sync.com"}
	}
	if cfg.Masking.Enabled {
		// Set default preserve values
		if !cfg.Masking.Geo.PreserveMetro && !cfg.Masking.Geo.PreserveZip && !cfg.Masking.Geo.PreserveCity {
			cfg.Masking.Geo.PreserveMetro = true
			cfg.Masking.Geo.PreserveZip = true
		}
	}

	// Create HTTP client with optimized transport for high-frequency API calls
	transport := &http.Transport{
		MaxIdleConns:        100,              // Allow more idle connections for connection reuse
		MaxIdleConnsPerHost: 10,               // Allow multiple connections per host
		IdleConnTimeout:     90 * time.Second, // Keep connections alive longer
		DisableCompression:  false,            // Enable compression to reduce bandwidth
		ForceAttemptHTTP2:   true,             // Use HTTP/2 when possible for better performance
	}

	return &Module{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout:   time.Duration(cfg.Timeout) * time.Millisecond,
			Transport: transport,
		},
		cache: &segmentCache{data: make(map[string]cacheEntry)},
	}, nil
}

// Config holds module configuration
type Config struct {
	Endpoint       string        `json:"endpoint"`
	AuthKey        string        `json:"auth_key"`
	Timeout        int           `json:"timeout_ms"`
	CacheTTL       int           `json:"cache_ttl_seconds"` // Cache segments for this many seconds
	AddToTargeting bool          `json:"add_to_targeting"`  // Add segments as individual targeting keys
	Masking        MaskingConfig `json:"masking"`           // Privacy masking configuration
}

// MaskingConfig controls what user data is masked before sending to Scope3
type MaskingConfig struct {
	Enabled bool                `json:"enabled"`
	Geo     GeoMaskingConfig    `json:"geo"`
	User    UserMaskingConfig   `json:"user"`
	Device  DeviceMaskingConfig `json:"device"`
}

// GeoMaskingConfig controls geographic data masking
type GeoMaskingConfig struct {
	PreserveMetro    bool `json:"preserve_metro"`     // DMA code (default: true)
	PreserveZip      bool `json:"preserve_zip"`       // Postal code (default: true)
	PreserveCity     bool `json:"preserve_city"`      // City name (default: false)
	LatLongPrecision int  `json:"lat_long_precision"` // Decimal places for lat/long (0-4, default: 2)
}

// UserMaskingConfig controls user data masking
type UserMaskingConfig struct {
	PreserveEids []string `json:"preserve_eids"` // List of EID sources to preserve
}

// DeviceMaskingConfig controls device data masking
type DeviceMaskingConfig struct {
	PreserveMobileIds bool `json:"preserve_mobile_ids"` // Keep mobile advertising IDs (default: false)
}

// cacheEntry represents a cached segment response
type cacheEntry struct {
	segments  []string
	timestamp time.Time
}

// segmentCache provides thread-safe caching of segment data
type segmentCache struct {
	mu   sync.RWMutex
	data map[string]cacheEntry
}

type userExt struct {
	Eids           []openrtb2.EID `json:"eids"`
	RampID         string         `json:"rampid"`
	LiverampIDL    string         `json:"liveramp_idl"`
	ATSEnvelope    string         `json:"ats_envelope"`
	RampIDEnvelope string         `json:"rampId_envelope"`
}

func (c *segmentCache) get(key string, ttl time.Duration) ([]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.data[key]
	if !exists || time.Since(entry.timestamp) > ttl {
		return nil, false
	}
	return entry.segments, true
}

func (c *segmentCache) set(key string, segments []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = cacheEntry{
		segments:  segments,
		timestamp: time.Now(),
	}
}

// Module implements the Scope3 RTD module
type Module struct {
	cfg        Config
	httpClient *http.Client
	cache      *segmentCache
}

// HandleEntrypointHook initializes the module context with a sync.Map for storing segments
func (m *Module) HandleEntrypointHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	// Initialize module context with sync.Map for thread-safe segment storage
	return hookstage.HookResult[hookstage.EntrypointPayload]{
		ModuleContext: hookstage.ModuleContext{
			"segments": &sync.Map{},
		},
	}, nil
}

// HandleRawAuctionHook is called early in the auction to fetch Scope3 data
func (m *Module) HandleRawAuctionHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.RawAuctionRequestPayload,
) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	// Parse the OpenRTB request
	var bidRequest openrtb2.BidRequest
	if err := jsonutil.Unmarshal(payload, &bidRequest); err != nil {
		return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, nil
	}

	// Call Scope3 API
	segments, err := m.fetchScope3Segments(ctx, &bidRequest)
	if err != nil {
		// Log error but don't fail the auction
		return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{
			AnalyticsTags: hookanalytics.Analytics{
				Activities: []hookanalytics.Activity{{
					Name:   "scope3_fetch",
					Status: hookanalytics.ActivityStatusError,
					Results: []hookanalytics.Result{{
						Status: hookanalytics.ResultStatusError,
						Values: map[string]interface{}{"error": err.Error()},
					}},
				}},
			},
		}, nil
	}

	// Store segments in module context
	if segmentStore, ok := miCtx.ModuleContext["segments"].(*sync.Map); ok {
		segmentStore.Store("segments", segments)
	}

	// Store segments for later use - no mutation needed at this stage
	changeSet := hookstage.ChangeSet[hookstage.RawAuctionRequestPayload]{}

	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{
		ChangeSet: changeSet,
		AnalyticsTags: hookanalytics.Analytics{
			Activities: []hookanalytics.Activity{{
				Name:   "scope3_fetch",
				Status: hookanalytics.ActivityStatusSuccess,
				Results: []hookanalytics.Result{{
					Status: hookanalytics.ResultStatusModify,
					Values: map[string]interface{}{
						"segments": segments,
						"count":    len(segments),
					},
				}},
			}},
		},
	}, nil
}

// HandleAuctionResponseHook adds targeting data to the auction response
func (m *Module) HandleAuctionResponseHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.AuctionResponsePayload,
) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	// Retrieve segments from module context
	var segments []string
	if segmentStore, ok := miCtx.ModuleContext["segments"].(*sync.Map); ok {
		if val, ok := segmentStore.Load("segments"); ok {
			segments = val.([]string)
		}
	}

	if len(segments) == 0 {
		return hookstage.HookResult[hookstage.AuctionResponsePayload]{}, nil
	}

	// Add segments to the auction response
	changeSet := hookstage.ChangeSet[hookstage.AuctionResponsePayload]{}
	changeSet.AddMutation(
		func(payload hookstage.AuctionResponsePayload) (hookstage.AuctionResponsePayload, error) {
			// Add Scope3 segments to the response ext so publisher can use them
			if payload.BidResponse.Ext == nil {
				payload.BidResponse.Ext = json.RawMessage("{}")
			}

			var extMap map[string]interface{}
			if err := jsonutil.Unmarshal(payload.BidResponse.Ext, &extMap); err != nil {
				extMap = make(map[string]interface{})
			}

			// Add segments as individual targeting keys for GAM integration
			if m.cfg.AddToTargeting {
				if prebidMap, ok := extMap["prebid"].(map[string]interface{}); ok {
					if targetingMap, ok := prebidMap["targeting"].(map[string]interface{}); ok {
						// Add each segment as individual targeting key
						for _, segment := range segments {
							targetingMap[segment] = "true"
						}
					} else {
						// Create targeting map with individual segment keys
						newTargeting := make(map[string]interface{})
						for _, segment := range segments {
							newTargeting[segment] = "true"
						}
						prebidMap["targeting"] = newTargeting
					}
				} else {
					// Create prebid map with targeting
					newTargeting := make(map[string]interface{})
					for _, segment := range segments {
						newTargeting[segment] = "true"
					}
					extMap["prebid"] = map[string]interface{}{
						"targeting": newTargeting,
					}
				}
			}

			// Always add to a dedicated scope3 section for publisher flexibility
			extMap["scope3"] = map[string]interface{}{
				"segments": segments,
			}

			extResp, err := jsonutil.Marshal(extMap)
			if err == nil {
				payload.BidResponse.Ext = extResp
			}

			return payload, nil
		},
		hookstage.MutationUpdate,
		"ext",
	)

	return hookstage.HookResult[hookstage.AuctionResponsePayload]{
		ChangeSet: changeSet,
	}, nil
}

// fetchScope3Segments calls the Scope3 API and extracts segments
func (m *Module) fetchScope3Segments(ctx context.Context, bidRequest *openrtb2.BidRequest) ([]string, error) {
	// Create cache key based on relevant user identifiers and site context
	cacheKey := m.createCacheKey(bidRequest)

	// Check cache first
	if segments, found := m.cache.get(cacheKey, time.Duration(m.cfg.CacheTTL)*time.Second); found {
		return segments, nil
	}

	// Apply privacy masking before sending to Scope3
	requestToSend := bidRequest
	if m.cfg.Masking.Enabled {
		requestToSend = m.maskBidRequest(bidRequest)
	}

	// Marshal the (potentially masked) bid request
	requestBody, err := jsonutil.Marshal(requestToSend)
	if err != nil {
		return nil, err
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", m.cfg.Endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-scope3-auth", m.cfg.AuthKey)

	// Make the request
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scope3 returned status %d", resp.StatusCode)
	}

	// Parse response
	var scope3Resp Scope3Response
	if err = json.NewDecoder(resp.Body).Decode(&scope3Resp); err != nil {
		return nil, err
	}

	// Extract unique segments (exclude destination)
	segmentMap := make(map[string]struct{})
	for _, data := range scope3Resp.Data {
		// Extract actual segments from impression-level data
		for _, imp := range data.Imp {
			if imp.Ext != nil && imp.Ext.Scope3 != nil {
				for _, segment := range imp.Ext.Scope3.Segments {
					segmentMap[segment.ID] = struct{}{}
				}
			}
		}
	}

	// Convert to slice
	segments := make([]string, 0, len(segmentMap))
	for segment := range segmentMap {
		segments = append(segments, segment)
	}

	// Cache the result
	m.cache.set(cacheKey, segments)

	return segments, nil
}

// createCacheKey generates a cache key based on user identifiers and site context
func (m *Module) createCacheKey(bidRequest *openrtb2.BidRequest) string {
	hasher := md5.New()

	// Include site/app information
	if bidRequest.Site != nil {
		hasher.Write([]byte(bidRequest.Site.Domain))
		hasher.Write([]byte(bidRequest.Site.Page))
	}
	if bidRequest.App != nil {
		hasher.Write([]byte(bidRequest.App.Bundle))
	}

	// Include user identifiers if available
	if bidRequest.User != nil && bidRequest.User.Ext != nil {
		var userExtension userExt
		if err := jsonutil.Unmarshal(bidRequest.User.Ext, &userExtension); err == nil {
			// Include LiveRamp identifiers
			for _, eid := range userExtension.Eids {
				if eid.Source == "liveramp.com" && len(eid.UIDs) > 0 {
					hasher.Write([]byte("rampid:" + eid.UIDs[0].ID))
				}
			}

			// Include other identifier types
			if userExtension.RampID != "" {
				hasher.Write([]byte("rampid:" + userExtension.RampID))
			}
			if userExtension.LiverampIDL != "" {
				hasher.Write([]byte("ats:" + userExtension.LiverampIDL))
			}

			// Include user ID if available
			if bidRequest.User != nil && bidRequest.User.ID != "" {
				hasher.Write([]byte("userid:" + bidRequest.User.ID))
			}
		}
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

// Response types for Scope3 API
type Scope3Response struct {
	Data []Scope3Data `json:"data"`
}

type Scope3Data struct {
	Destination string          `json:"destination"`
	Imp         []Scope3ImpData `json:"imp"`
}

type Scope3ImpData struct {
	ID  string     `json:"id"`
	Ext *Scope3Ext `json:"ext,omitempty"`
}

type Scope3Ext struct {
	Scope3 *Scope3ExtData `json:"scope3"`
}

type Scope3ExtData struct {
	Segments []Scope3Segment `json:"segments"`
}

type Scope3Segment struct {
	ID string `json:"id"`
}
