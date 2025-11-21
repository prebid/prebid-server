// Package scope3 implements a Prebid Server module for Scope3 RTD
package scope3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/coocood/freecache"
	"github.com/golang/glog"
	jsoniter "github.com/json-iterator/go"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/prebid/prebid-server/v3/util/iterutil"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/tidwall/sjson"
)

// Builder is the entry point for the module
// This is called by Prebid Server to initialize the module
func Builder(config json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	var cfg Config
	if err := jsonutil.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultScope3RTDURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 1000 // 1000ms default
	}
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 60 // 60 seconds default
	}
	if cfg.CacheSize == 0 {
		cfg.CacheSize = 10 * 1024 * 1024 // 10MB
	}

	// Set masking defaults and validate configuration
	if cfg.Masking.Enabled {
		// Validate and set geo precision (max 4 decimal places for privacy)
		if cfg.Masking.Geo.LatLongPrecision == 0 {
			cfg.Masking.Geo.LatLongPrecision = 2 // 2 decimal places default (~1.1km precision)
		} else if cfg.Masking.Geo.LatLongPrecision > 4 {
			return nil, errors.New("lat_long_precision cannot exceed 4 decimal places for privacy protection")
		} else if cfg.Masking.Geo.LatLongPrecision < 0 {
			return nil, errors.New("lat_long_precision cannot be negative")
		}

		// Set default EID allowlist if empty
		if len(cfg.Masking.User.PreserveEids) == 0 {
			// Default to preserving common identity providers
			cfg.Masking.User.PreserveEids = []string{"liveramp.com", "uidapi.com", "id5-sync.com"}
		}

		// Set default preserve values for geo fields
		if !cfg.Masking.Geo.PreserveMetro && !cfg.Masking.Geo.PreserveZip && !cfg.Masking.Geo.PreserveCity {
			cfg.Masking.Geo.PreserveMetro = true
			cfg.Masking.Geo.PreserveZip = true
		}
	}

	return &Module{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout:   time.Duration(cfg.Timeout) * time.Millisecond,
			Transport: deps.HTTPClient.Transport,
		},
		cache: freecache.NewCache(cfg.CacheSize),
		sha256Pool: &sync.Pool{
			New: func() any {
				return sha256.New()
			},
		},
	}, nil
}

const (
	// keys for miCtx
	asyncRequestKey      = "scope3.AsyncRequest"
	scope3MacroKey       = "scope3_macro"
	scope3MacroSeparator = ";"
)

var scope3MacroKeyPlusSeparator = scope3MacroKey + scope3MacroSeparator

const DefaultScope3RTDURL = "https://rtdp.scope3.com/prebid/prebid"

var (
	// Declare hooks
	_ hookstage.Entrypoint              = (*Module)(nil)
	_ hookstage.ProcessedAuctionRequest = (*Module)(nil)
	_ hookstage.AuctionResponse         = (*Module)(nil)
)

// Config holds module configuration
type Config struct {
	Endpoint                  string        `json:"endpoint"`
	AuthKey                   string        `json:"auth_key"`
	Timeout                   int           `json:"timeout_ms"`
	CacheTTL                  int           `json:"cache_ttl_seconds"`            // Cache segments for this many seconds
	CacheSize                 int           `json:"cache_size"`                   // Maximum size of segment cache in bytes
	AddToTargeting            bool          `json:"add_to_targeting"`             // Add segments as individual targeting keys
	AddScope3TargetingSection bool          `json:"add_scope3_targeting_section"` // Add segments as individual targeting keys in Scope3 targeting section
	Masking                   MaskingConfig `json:"masking"`                      // Privacy masking configuration
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

type userExt struct {
	Eids           []openrtb2.EID `json:"eids"`
	RampID         string         `json:"rampid"`
	LiverampIDL    string         `json:"liveramp_idl"`
	ATSEnvelope    string         `json:"ats_envelope"`
	RampIDEnvelope string         `json:"rampId_envelope"`
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
	Macro    string          `json:"macro"`
}

type Scope3Segment struct {
	ID string `json:"id"`
}

// Module implements the Scope3 RTD module
type Module struct {
	cfg        Config
	httpClient *http.Client
	cache      *freecache.Cache
	// sha256Pool provides a pool of reusable SHA-256 hash instances for performance
	sha256Pool *sync.Pool
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
			asyncRequestKey: m.NewAsyncRequest(payload.Request),
		},
	}, nil
}

// HandleRawAuctionHook is called early in the auction to fetch Scope3 data
func (m *Module) HandleProcessedAuctionHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	var ret hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]
	analyticsNamePrefix := "HandleProcessedAuctionHook."

	asyncRequest, ok := miCtx.ModuleContext[asyncRequestKey].(*AsyncRequest)
	if !ok {
		// Log error but don't fail the auction
		ret.AnalyticsTags = hookanalytics.Analytics{
			Activities: []hookanalytics.Activity{{
				Name:   analyticsNamePrefix + asyncRequestKey,
				Status: hookanalytics.ActivityStatusError,
				Results: []hookanalytics.Result{{
					Status: hookanalytics.ResultStatusError,
					Values: map[string]interface{}{"error": "failed to get async request from module context"},
				}},
			}},
		}
		return ret, nil
	}

	// Start async request to Scope3
	asyncRequest.fetchScope3SegmentsAsync(payload.Request.BidRequest)

	return ret, nil
}

// HandleAuctionResponseHook adds targeting data to the auction response
func (m *Module) HandleAuctionResponseHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.AuctionResponsePayload,
) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	analyticsNamePrefix := "HandleAuctionResponseHook."
	var ret hookstage.HookResult[hookstage.AuctionResponsePayload]
	asyncRequest, ok := miCtx.ModuleContext[asyncRequestKey].(*AsyncRequest)
	if !ok {
		// Log error but don't fail the auction
		ret.AnalyticsTags = hookanalytics.Analytics{
			Activities: []hookanalytics.Activity{{
				Name:   analyticsNamePrefix + asyncRequestKey,
				Status: hookanalytics.ActivityStatusError,
				Results: []hookanalytics.Result{{
					Status: hookanalytics.ResultStatusError,
					Values: map[string]interface{}{"error": "failed to get async request from module context"},
				}},
			}},
		}
		return ret, nil
	}
	// Ensure we cancel the request context always to free resources
	defer asyncRequest.Cancel()

	// Check if a request was made
	if asyncRequest.Done == nil {
		return ret, nil
	}

	// Wait for the async request to complete
	select {
	case <-asyncRequest.Done:
		// Continue with processing
	case <-ctx.Done():
		return ret, nil // Context cancelled, exit gracefully
	}

	// Get results
	segments, err := asyncRequest.Segments, asyncRequest.Err
	if err != nil {
		// Log error but don't fail the auction
		ret.AnalyticsTags = hookanalytics.Analytics{
			Activities: []hookanalytics.Activity{{
				Name:   analyticsNamePrefix + "scope3_fetch",
				Status: hookanalytics.ActivityStatusError,
				Results: []hookanalytics.Result{{
					Status: hookanalytics.ResultStatusError,
					Values: map[string]interface{}{"error": err.Error()},
				}},
			}},
		}
		return ret, nil
	}

	if len(segments) == 0 {
		return ret, nil
	}

	// Add segments to the auction response
	ret.ChangeSet.AddMutation(
		func(payload hookstage.AuctionResponsePayload) (hookstage.AuctionResponsePayload, error) {
			// Add segments as individual targeting keys for GAM integration
			if m.cfg.AddToTargeting {
				// Add each segment as individual targeting key
				for _, segment := range segments {
					if strings.HasPrefix(segment, scope3MacroKeyPlusSeparator) {
						macroKeyVal := strings.Split(segment, scope3MacroSeparator)
						if len(macroKeyVal) != 2 {
							continue
						}
						newPayload, err := sjson.SetBytes(payload.BidResponse.Ext, "prebid.targeting."+macroKeyVal[0], macroKeyVal[1])
						if err != nil {
							return payload, err
						}
						payload.BidResponse.Ext = newPayload
					} else {
						newPayload, err := sjson.SetBytes(payload.BidResponse.Ext, "prebid.targeting."+segment, "true")
						if err != nil {
							return payload, err
						}
						payload.BidResponse.Ext = newPayload
					}
				}
			}

			// Add to a dedicated scope3 section for publisher flexibility when configured
			if m.cfg.AddScope3TargetingSection {
				newPayload, err := sjson.SetBytes(payload.BidResponse.Ext, "scope3.segments", segments)
				if err != nil {
					return payload, err
				}
				payload.BidResponse.Ext = newPayload
			}

			// also add to seatbid[].bid[]
			for seatBid := range iterutil.SlicePointerValues(payload.BidResponse.SeatBid) {
				for bid := range iterutil.SlicePointerValues(seatBid.Bid) {
					// Add segments as individual targeting keys for GAM integration
					if m.cfg.AddToTargeting {
						for _, segment := range segments {
							if strings.HasPrefix(segment, scope3MacroKeyPlusSeparator) {
								macroKeyVal := strings.Split(segment, scope3MacroSeparator)
								if len(macroKeyVal) != 2 {
									continue
								}
								newPayload, err := sjson.SetBytes(bid.Ext, "prebid.targeting."+macroKeyVal[0], macroKeyVal[1])
								if err != nil {
									return payload, err
								}
								bid.Ext = newPayload
							} else {
								newPayload, err := sjson.SetBytes(bid.Ext, "prebid.targeting."+segment, "true")
								if err != nil {
									return payload, err
								}
								bid.Ext = newPayload
							}
						}
					}

					// Always add to a dedicated scope3 section for publisher flexibility
					if m.cfg.AddScope3TargetingSection {
						newPayload, err := sjson.SetBytes(bid.Ext, "scope3.segments", segments)
						if err != nil {
							return payload, err
						}
						bid.Ext = newPayload
					}
				}
			}

			return payload, nil
		},
		hookstage.MutationUpdate,
		"ext",
	)

	return ret, nil
}

// fetchScope3Segments calls the Scope3 API and extracts segments
func (m *Module) fetchScope3Segments(ctx context.Context, bidRequest *openrtb2.BidRequest) ([]string, error) {
	// Create cache key based on relevant user identifiers and site context
	cacheKey := []byte(m.createCacheKey(bidRequest))

	// Check cache first
	if segments, err := m.cache.Get(cacheKey); err == nil {
		return strings.Split(string(segments), ","), nil
	}

	// Apply privacy masking before sending to Scope3
	requestToSend := bidRequest
	if m.cfg.Masking.Enabled {
		maskedRequest := m.maskBidRequest(bidRequest)
		if maskedRequest == nil {
			// Masking failed - don't send request to prevent data leakage
			return nil, errors.New("failed to mask bid request for privacy protection")
		}
		requestToSend = maskedRequest
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
	if err = jsoniter.ConfigCompatibleWithStandardLibrary.NewDecoder(resp.Body).Decode(&scope3Resp); err != nil {
		return nil, err
	}

	// Extract unique segments (exclude destination)
	segmentMap := make(map[string]bool)
	var macro string
	for data := range iterutil.SlicePointerValues(scope3Resp.Data) {
		// Extract actual segments from impression-level data
		for imp := range iterutil.SlicePointerValues(data.Imp) {
			if imp.Ext != nil && imp.Ext.Scope3 != nil {
				if imp.Ext.Scope3.Macro != "" {
					macro = imp.Ext.Scope3.Macro
				}
				for segment := range iterutil.SlicePointerValues(imp.Ext.Scope3.Segments) {
					segmentMap[segment.ID] = true
				}
			}
		}
	}

	// Convert to slice
	segments := slices.AppendSeq(make([]string, 0, len(segmentMap)), maps.Keys(segmentMap))
	if macro != "" {
		segments = append(segments, scope3MacroKeyPlusSeparator+macro)
	}

	// Cache the result
	err = m.cache.Set(cacheKey, []byte(strings.Join(segments, ",")), m.cfg.CacheTTL)
	if err != nil {
		glog.Infof("could not set segments in cache: %v", err)
	}

	return segments, nil
}

// createCacheKey generates a cache key based on non-sensitive context and identifiers
// Note: Uses only privacy-safe identifiers to prevent correlation attacks
func (m *Module) createCacheKey(bidRequest *openrtb2.BidRequest) string {
	hasher := m.sha256Pool.Get().(hash.Hash)
	hasher.Reset()
	defer m.sha256Pool.Put(hasher)

	// Include site/app information (not sensitive)
	if bidRequest.Site != nil {
		hasher.Write([]byte("site:" + bidRequest.Site.Domain))
		if bidRequest.Site.Page != "" {
			hasher.Write([]byte("page:" + bidRequest.Site.Page))
		}
	}
	if bidRequest.App != nil {
		hasher.Write([]byte("app:" + bidRequest.App.Bundle))
	}

	// Include user identifiers for per-user caching
	hasPrivacySafeID := false
	if bidRequest.User != nil && bidRequest.User.Ext != nil {
		var userExtension userExt
		if err := jsonutil.Unmarshal(bidRequest.User.Ext, &userExtension); err == nil {
			// Include LiveRamp identifiers (these are privacy-safe for caching)
			for eid := range iterutil.SlicePointerValues(userExtension.Eids) {
				if eid.Source == "liveramp.com" && len(eid.UIDs) > 0 {
					hasher.Write([]byte("eid:rampid:" + eid.UIDs[0].ID))
					hasPrivacySafeID = true
				}
			}

			// Include other privacy-safe identifier types
			if userExtension.RampID != "" {
				hasher.Write([]byte("eid:rampid:" + userExtension.RampID))
				hasPrivacySafeID = true
			}
			if userExtension.LiverampIDL != "" {
				hasher.Write([]byte("eid:ats:" + userExtension.LiverampIDL))
				hasPrivacySafeID = true
			}
		}
	}

	// If no privacy-safe identifiers are available, use hashed user.id for per-user caching
	if !hasPrivacySafeID && bidRequest.User != nil && bidRequest.User.ID != "" {
		userHasher := m.sha256Pool.Get().(hash.Hash)
		userHasher.Reset()
		defer m.sha256Pool.Put(userHasher)

		userHasher.Write([]byte("user_id:" + bidRequest.User.ID))
		hasher.Write([]byte("hashed_user_id:" + hex.EncodeToString(userHasher.Sum(nil))))
	}

	return hex.EncodeToString(hasher.Sum(nil))
}
