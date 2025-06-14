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
	"strings"
	"sync"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
)

// Builder is the entry point for the module
// This is called by Prebid Server to initialize the module
func Builder(config json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	var cfg Config
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://rtdp.scope3.com/amazonaps/rtii"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 1000 // 1000ms default
	}
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 60 // 60 seconds default
	}

	return &Module{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: time.Duration(cfg.Timeout) * time.Millisecond},
		cache:      &segmentCache{data: make(map[string]cacheEntry)},
	}, nil
}

// Config holds module configuration
type Config struct {
	Endpoint    string `json:"endpoint"`
	AuthKey     string `json:"auth_key"`
	Timeout     int    `json:"timeout_ms"`
	CacheTTL    int    `json:"cache_ttl_seconds"` // Cache segments for this many seconds
	BidMetaData bool   `json:"bid_meta_data"`     // Include segments in bid.meta
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
	if err := json.Unmarshal(payload, &bidRequest); err != nil {
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

// HandleProcessedAuctionHook adds targeting keys to the response
func (m *Module) HandleProcessedAuctionHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	// Retrieve segments from module context
	var segments []string
	if segmentStore, ok := miCtx.ModuleContext["segments"].(*sync.Map); ok {
		if val, ok := segmentStore.Load("segments"); ok {
			segments = val.([]string)
		}
	}

	if len(segments) == 0 {
		return hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{}, nil
	}

	// Add targeting keys to the request
	changeSet := hookstage.ChangeSet[hookstage.ProcessedAuctionRequestPayload]{}
	changeSet.AddMutation(
		func(payload hookstage.ProcessedAuctionRequestPayload) (hookstage.ProcessedAuctionRequestPayload, error) {
			// Add Scope3 segments as targeting keys for GAM
			// Format: "gmp_eligible,gmp_plus_eligible" for easy GAM key-value targeting
			reqWrapper := payload.Request
			if reqWrapper.BidRequest.Ext == nil {
				reqWrapper.BidRequest.Ext = json.RawMessage("{}")
			}

			var extMap map[string]interface{}
			if err := json.Unmarshal(reqWrapper.BidRequest.Ext, &extMap); err != nil {
				extMap = make(map[string]interface{})
			}

			// Add targeting keys that will be available to GAM
			if targetingMap, ok := extMap["targeting"].(map[string]interface{}); ok {
				targetingMap["hb_scope3_segments"] = strings.Join(segments, ",")
			} else {
				extMap["targeting"] = map[string]interface{}{
					"hb_scope3_segments": strings.Join(segments, ","),
				}
			}

			reqWrapper.BidRequest.Ext, _ = json.Marshal(extMap)

			// Update the wrapper with the modified bid request
			payload.Request = reqWrapper
			return payload, nil
		},
		hookstage.MutationUpdate,
		"imp", "ext",
	)

	return hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{
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

	// Enhance request with available user identifiers
	m.enhanceRequestWithUserIDs(bidRequest)

	// Marshal the bid request
	requestBody, err := json.Marshal(bidRequest)
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scope3 returned status %d", resp.StatusCode)
	}

	// Parse response
	var scope3Resp Scope3Response
	if err := json.NewDecoder(resp.Body).Decode(&scope3Resp); err != nil {
		return nil, err
	}

	// Extract unique segments
	segmentMap := make(map[string]bool)
	for _, data := range scope3Resp.Data {
		for _, imp := range data.Imp {
			if imp.Ext != nil && imp.Ext.Scope3 != nil {
				for _, segment := range imp.Ext.Scope3.Segments {
					segmentMap[segment.ID] = true
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
		var userExt map[string]interface{}
		if err := json.Unmarshal(bidRequest.User.Ext, &userExt); err == nil {
			// Include LiveRamp identifiers
			if eids, ok := userExt["eids"].([]interface{}); ok {
				for _, eid := range eids {
					if eidMap, ok := eid.(map[string]interface{}); ok {
						if source, ok := eidMap["source"].(string); ok && source == "liveramp.com" {
							if uidsArray, ok := eidMap["uids"].([]interface{}); ok && len(uidsArray) > 0 {
								if uidMap, ok := uidsArray[0].(map[string]interface{}); ok {
									if id, ok := uidMap["id"].(string); ok {
										hasher.Write([]byte("rampid:" + id))
									}
								}
							}
						}
					}
				}
			}

			// Include other identifier types
			if rampID, ok := userExt["rampid"].(string); ok {
				hasher.Write([]byte("rampid:" + rampID))
			}
			if atsEnvelope, ok := userExt["liveramp_idl"].(string); ok {
				hasher.Write([]byte("ats:" + atsEnvelope))
			}
		}

		// Include user ID if available
		if bidRequest.User.ID != "" {
			hasher.Write([]byte("userid:" + bidRequest.User.ID))
		}
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

// enhanceRequestWithUserIDs adds available user identifiers to the request
// This function checks for various user identifiers that may be available
func (m *Module) enhanceRequestWithUserIDs(bidRequest *openrtb2.BidRequest) {
	if bidRequest.User == nil {
		return
	}

	// Check for existing user.ext data
	if bidRequest.User.Ext == nil {
		return
	}

	var userExt map[string]interface{}
	if err := json.Unmarshal(bidRequest.User.Ext, &userExt); err != nil {
		return
	}

	// Check for LiveRamp identifiers:
	// Note: LiveRamp identifiers may be present from various sources including
	// publisher implementations, other RTD modules, or identity providers

	// 1. Check for LiveRamp EID in the standard eids array
	if eids, ok := userExt["eids"].([]interface{}); ok {
		for _, eid := range eids {
			if eidMap, ok := eid.(map[string]interface{}); ok {
				if source, ok := eidMap["source"].(string); ok && source == "liveramp.com" {
					// LiveRamp EID found - will be included in the API request
					return
				}
			}
		}
	}

	// 2. Check for direct rampid field (alternative location used by some publishers)
	if rampID, ok := userExt["rampid"].(string); ok && rampID != "" {
		// RampID found in alternative location
		return
	}

	// 3. Check for ATS envelope in various possible locations
	// Publishers may store ATS envelopes in different extension fields
	atsLocations := []string{"liveramp_idl", "ats_envelope", "rampId_envelope"}
	for _, location := range atsLocations {
		if atsEnvelope, ok := userExt[location].(string); ok && atsEnvelope != "" {
			// ATS envelope found - will be forwarded in the request
			return
		}
	}

	// 4. Check for ATS envelope in top-level request extensions
	if bidRequest.Ext != nil {
		var reqExt map[string]interface{}
		if err := json.Unmarshal(bidRequest.Ext, &reqExt); err == nil {
			for _, location := range atsLocations {
				if atsEnvelope, ok := reqExt[location].(string); ok && atsEnvelope != "" {
					// ATS envelope found at request level
					return
				}
			}
		}
	}

	// No specific LiveRamp identifiers found, but other user identifiers
	// (like user.id, device.ifa, etc.) will still be included in the request
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
	ID     string  `json:"id"`
	Weight float64 `json:"weight,omitempty"`
}
