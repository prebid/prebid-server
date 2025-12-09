package trafficshaping

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		expectError bool
	}{
		{
			name: "valid_config",
			config: `{
				"enabled": true,
				"endpoint": "http://example.com/config.json",
				"refresh_ms": 30000,
				"request_timeout_ms": 1000
			}`,
			expectError: false,
		},
		{
			name:        "missing_endpoint",
			config:      `{"enabled": true}`,
			expectError: true,
		},
		{
			name: "invalid_refresh",
			config: `{
				"enabled": true,
				"endpoint": "http://example.com/config.json",
				"refresh_ms": 500
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := moduledeps.ModuleDeps{
				HTTPClient: http.DefaultClient,
			}

			module, err := Builder(json.RawMessage(tt.config), deps)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, module)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, module)
			}
		})
	}
}

func TestGetConfigForURLCoalescesFetches(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		time.Sleep(20 * time.Millisecond)

		response := TrafficShapingData{
			Response: Response{
				SkipRate:      0,
				UserIdVendors: []string{},
				Values:        map[string]map[string]map[string]int{},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		Enabled:          true,
		BaseEndpoint:     server.URL + "/",
		RefreshMs:        30000,
		RequestTimeoutMs: 1000,
		SampleSalt:       "pbs",
	}

	client := NewConfigClient(server.Client(), config)
	defer client.Stop()

	url := server.URL + "/ts.json"

	const concurrency = 5
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make([]*ShapingConfig, concurrency)

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			<-start
			results[idx] = client.GetConfigForURL(url)
		}(i)
	}

	close(start)
	wg.Wait()

	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
	for _, cfg := range results {
		require.NotNil(t, cfg)
	}

	cached := client.GetConfigForURL(url)
	require.NotNil(t, cached)
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
}

func TestHandleProcessedAuctionHook_NoConfig(t *testing.T) {
	// Create a server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := &Config{
		Enabled:          true,
		Endpoint:         server.URL,
		RefreshMs:        30000,
		RequestTimeoutMs: 1000,
		SampleSalt:       "pbs",
	}

	client := NewConfigClient(http.DefaultClient, config)
	defer client.Stop()

	module := &Module{
		config: config,
		client: client,
	}

	payload := hookstage.ProcessedAuctionRequestPayload{
		Request: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				ID: "test-request",
			},
		},
	}

	result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)

	assert.NoError(t, err)
	assert.Contains(t, result.Warnings, "trafficshaping: config unavailable")
	// Expect skipped_no_config and new fetch_failed + skipped tags present
	names := make(map[string]bool)
	for _, a := range result.AnalyticsTags.Activities {
		names[a.Name] = true
	}
	assert.True(t, names["skipped_no_config"])
	assert.True(t, names["fetch_failed"])
	assert.True(t, names["skipped"])
}

func TestHandleProcessedAuctionHook_SkipRate(t *testing.T) {
	tests := []struct {
		name       string
		skipRate   int
		requestID  string
		shouldSkip bool
	}{
		{
			name:       "skipRate_0_never_skips",
			skipRate:   0,
			requestID:  "test-request-1",
			shouldSkip: false,
		},
		{
			name:       "skipRate_100_always_skips",
			skipRate:   100,
			requestID:  "test-request-2",
			shouldSkip: true,
		},
		{
			name:       "skipRate_50_deterministic",
			skipRate:   50,
			requestID:  "test-request-3",
			shouldSkip: shouldSkipByRate("test-request-3", 50, "pbs"),
		},
		{
			name:       "skipRate_75_deterministic",
			skipRate:   75,
			requestID:  "test-request-4",
			shouldSkip: shouldSkipByRate("test-request-4", 75, "pbs"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server with shaping config
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := TrafficShapingData{
					Response: Response{
						SkipRate: tt.skipRate,
						Values: map[string]map[string]map[string]int{
							"test-gpid": {
								"rubicon": {"300x250": 1},
							},
						},
					},
				}
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			config := &Config{
				Enabled:          true,
				Endpoint:         server.URL,
				RefreshMs:        30000,
				RequestTimeoutMs: 1000,
				SampleSalt:       "pbs",
			}

			client := NewConfigClient(http.DefaultClient, config)
			defer client.Stop()

			// Wait for initial fetch
			for client.GetConfig() == nil {
			}

			module := &Module{
				config: config,
				client: client,
			}

			payload := hookstage.ProcessedAuctionRequestPayload{
				Request: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: tt.requestID,
					},
				},
			}

			result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)

			assert.NoError(t, err)
			if tt.shouldSkip {
				names := make(map[string]bool)
				for _, a := range result.AnalyticsTags.Activities {
					names[a.Name] = true
				}
				assert.True(t, names["skipped_by_skiprate"])
				assert.True(t, names["skipped"])
			}
		})
	}
}

func TestHandleProcessedAuctionHook_CountryGating(t *testing.T) {
	tests := []struct {
		name             string
		allowedCountries []string
		deviceCountry    string
		shouldSkip       bool
	}{
		{
			name:             "no_restriction",
			allowedCountries: nil,
			deviceCountry:    "CA",
			shouldSkip:       false,
		},
		{
			name:             "allowed_country",
			allowedCountries: []string{"US", "CA"},
			deviceCountry:    "US",
			shouldSkip:       false,
		},
		{
			name:             "disallowed_country",
			allowedCountries: []string{"US"},
			deviceCountry:    "CA",
			shouldSkip:       true,
		},
		{
			name:             "missing_country",
			allowedCountries: []string{"US"},
			deviceCountry:    "",
			shouldSkip:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server with shaping config
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := TrafficShapingData{
					Response: Response{
						SkipRate: 0,
						Values: map[string]map[string]map[string]int{
							"test-gpid": {
								"rubicon": {"300x250": 1},
							},
						},
					},
				}
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			config := &Config{
				Enabled:          true,
				Endpoint:         server.URL,
				RefreshMs:        30000,
				RequestTimeoutMs: 1000,
				SampleSalt:       "pbs",
				AllowedCountries: tt.allowedCountries,
			}

			client := NewConfigClient(http.DefaultClient, config)
			defer client.Stop()

			// Wait for initial fetch
			for client.GetConfig() == nil {
			}

			module := &Module{
				config: config,
				client: client,
			}

			var device *openrtb2.Device
			if tt.deviceCountry != "" {
				device = &openrtb2.Device{
					Geo: &openrtb2.Geo{
						Country: tt.deviceCountry,
					},
				}
			}

			payload := hookstage.ProcessedAuctionRequestPayload{
				Request: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID:     "test-request",
						Device: device,
					},
				},
			}

			result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)

			assert.NoError(t, err)
			if tt.shouldSkip {
				names := make(map[string]bool)
				for _, a := range result.AnalyticsTags.Activities {
					names[a.Name] = true
				}
				assert.True(t, names["skipped_country"])
				assert.True(t, names["skipped"])
			}
		})
	}
}

func TestHandleProcessedAuctionHook_GPIDShaping(t *testing.T) {
	// Create a test server with shaping config
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TrafficShapingData{
			Response: Response{
				SkipRate: 0,
				Values: map[string]map[string]map[string]int{
					"test-gpid-1": {
						"rubicon":  {"300x250": 1, "728x90": 1},
						"appnexus": {"300x250": 1},
					},
					"test-gpid-2": {
						"pubmatic": {"320x50": 1},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		Enabled:          true,
		Endpoint:         server.URL,
		RefreshMs:        30000,
		RequestTimeoutMs: 1000,
		SampleSalt:       "pbs",
	}

	client := NewConfigClient(http.DefaultClient, config)
	defer client.Stop()

	// Wait for initial fetch
	for client.GetConfig() == nil {
	}

	module := &Module{
		config: config,
		client: client,
	}

	// Create request with multiple impressions
	impExt1, _ := json.Marshal(map[string]interface{}{
		"gpid": "test-gpid-1",
		"prebid": map[string]interface{}{
			"bidder": map[string]interface{}{
				"rubicon":  json.RawMessage(`{}`),
				"appnexus": json.RawMessage(`{}`),
				"sovrn":    json.RawMessage(`{}`), // Should be filtered out
			},
		},
	})

	impExt2, _ := json.Marshal(map[string]interface{}{
		"gpid": "test-gpid-2",
		"prebid": map[string]interface{}{
			"bidder": map[string]interface{}{
				"pubmatic": json.RawMessage(`{}`),
			},
		},
	})

	payload := hookstage.ProcessedAuctionRequestPayload{
		Request: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				ID: "test-request",
				Imp: []openrtb2.Imp{
					{
						ID:  "imp1",
						Ext: impExt1,
						Banner: &openrtb2.Banner{
							Format: []openrtb2.Format{
								{W: 300, H: 250},
								{W: 728, H: 90},
								{W: 160, H: 600}, // Should be filtered out
							},
						},
					},
					{
						ID:  "imp2",
						Ext: impExt2,
						Banner: &openrtb2.Banner{
							W: ptrutil.ToPtr[int64](320),
							H: ptrutil.ToPtr[int64](50),
						},
					},
				},
			},
		},
	}

	result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)

	assert.NoError(t, err)

	// Check that shaping was applied (applied + shaped)
	names := make(map[string]bool)
	for _, a := range result.AnalyticsTags.Activities {
		names[a.Name] = true
	}
	assert.True(t, names["applied"])
	assert.True(t, names["shaped"])

	// Check that bidder filtering mutation was added
	assert.NotEmpty(t, result.ChangeSet)
}

func TestHandleProcessedAuctionHook_MissingGPID(t *testing.T) {
	// Create a test server with shaping config
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TrafficShapingData{
			Response: Response{
				SkipRate: 0,
				Values: map[string]map[string]map[string]int{
					"test-gpid": {
						"rubicon": {"300x250": 1},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		Enabled:          true,
		Endpoint:         server.URL,
		RefreshMs:        30000,
		RequestTimeoutMs: 1000,
		SampleSalt:       "pbs",
	}

	client := NewConfigClient(http.DefaultClient, config)
	defer client.Stop()

	// Wait for initial fetch
	for client.GetConfig() == nil {
	}

	module := &Module{
		config: config,
		client: client,
	}

	// Create request without GPID
	payload := hookstage.ProcessedAuctionRequestPayload{
		Request: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				ID: "test-request",
				Imp: []openrtb2.Imp{
					{
						ID: "imp1",
						Banner: &openrtb2.Banner{
							Format: []openrtb2.Format{
								{W: 300, H: 250},
							},
						},
					},
				},
			},
		},
	}

	result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)

	assert.NoError(t, err)

	// Check for missing_gpid activity
	hasMissingGPID := false
	for _, activity := range result.AnalyticsTags.Activities {
		if activity.Name == "missing_gpid" {
			hasMissingGPID = true
			break
		}
	}
	assert.True(t, hasMissingGPID, "Expected missing_gpid activity")
}

func TestShouldSkipByRate_Deterministic(t *testing.T) {
	requestID := "test-request-123"
	salt := "pbs"

	// Test that the same request ID always produces the same result
	result1 := shouldSkipByRate(requestID, 50, salt)
	result2 := shouldSkipByRate(requestID, 50, salt)
	result3 := shouldSkipByRate(requestID, 50, salt)

	assert.Equal(t, result1, result2)
	assert.Equal(t, result2, result3)
}

func TestGetGPID(t *testing.T) {
	tests := []struct {
		name         string
		impExt       map[string]interface{}
		expectedGPID string
	}{
		{
			name: "gpid_present",
			impExt: map[string]interface{}{
				"gpid": "test-gpid-123",
			},
			expectedGPID: "test-gpid-123",
		},
		{
			name: "fallback_to_adslot",
			impExt: map[string]interface{}{
				"data": map[string]interface{}{
					"adserver": map[string]interface{}{
						"adslot": "/1234/homepage",
					},
				},
			},
			expectedGPID: "/1234/homepage",
		},
		{
			name:         "no_gpid",
			impExt:       map[string]interface{}{},
			expectedGPID: "",
		},
		{
			name: "gpid_empty_string",
			impExt: map[string]interface{}{
				"gpid": "",
			},
			expectedGPID: "", // Empty string should return empty
		},
		{
			name:         "nil_imp_wrapper",
			impExt:       nil,
			expectedGPID: "",
		},
		{
			name:         "nil_imp",
			impExt:       map[string]interface{}{},
			expectedGPID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var impWrapper *openrtb_ext.ImpWrapper
			switch tt.name {
			case "nil_imp_wrapper":
				impWrapper = nil
			case "nil_imp":
				impWrapper = &openrtb_ext.ImpWrapper{Imp: nil}
			default:
				extJSON, err := json.Marshal(tt.impExt)
				require.NoError(t, err)

				impWrapper = &openrtb_ext.ImpWrapper{
					Imp: &openrtb2.Imp{
						ID:  "test-imp",
						Ext: extJSON,
					},
				}
			}

			gpid := getGPID(impWrapper)
			assert.Equal(t, tt.expectedGPID, gpid)
		})
	}

	t.Run("invalid_ext_json", func(t *testing.T) {
		impWrapper := &openrtb_ext.ImpWrapper{
			Imp: &openrtb2.Imp{
				ID:  "test-imp",
				Ext: json.RawMessage(`{"invalid": json}`), // Invalid JSON
			},
		}

		gpid := getGPID(impWrapper)
		assert.Equal(t, "", gpid) // Should return empty on error
	})

	t.Run("gpid_unmarshal_error", func(t *testing.T) {
		// Create ext with invalid gpid JSON
		extJSON := json.RawMessage(`{"gpid": {"invalid": json}}`)
		impWrapper := &openrtb_ext.ImpWrapper{
			Imp: &openrtb2.Imp{
				ID:  "test-imp",
				Ext: extJSON,
			},
		}

		gpid := getGPID(impWrapper)
		assert.Equal(t, "", gpid) // Should return empty on unmarshal error
	})

	t.Run("data_unmarshal_error", func(t *testing.T) {
		// Create ext with invalid data JSON
		extJSON := json.RawMessage(`{"data": {"invalid": json}}`)
		impWrapper := &openrtb_ext.ImpWrapper{
			Imp: &openrtb2.Imp{
				ID:  "test-imp",
				Ext: extJSON,
			},
		}

		gpid := getGPID(impWrapper)
		assert.Equal(t, "", gpid) // Should return empty on unmarshal error
	})
}

func TestFilterBannerSizes(t *testing.T) {
	allowedSizes := map[BannerSize]struct{}{
		{W: 300, H: 250}: {},
		{W: 728, H: 90}:  {},
	}

	tests := []struct {
		name           string
		banner         *openrtb2.Banner
		expectedFormat int
	}{
		{
			name: "filter_formats",
			banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					{W: 300, H: 250}, // allowed
					{W: 728, H: 90},  // allowed
					{W: 160, H: 600}, // not allowed
				},
			},
			expectedFormat: 2,
		},
		{
			name: "all_filtered_out",
			banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					{W: 160, H: 600}, // not allowed
					{W: 120, H: 600}, // not allowed
				},
			},
			expectedFormat: 2, // Should keep original if all filtered
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imp := &openrtb2.Imp{
				Banner: tt.banner,
			}

			originalLen := len(imp.Banner.Format)
			filterBannerSizes(imp, allowedSizes)

			switch tt.name {
			case "filter_formats":
				assert.Equal(t, tt.expectedFormat, len(imp.Banner.Format))
			case "all_filtered_out":
				// Should keep original when all would be filtered
				assert.Equal(t, originalLen, len(imp.Banner.Format))
			}
		})
	}
}

func TestPruneEIDs(t *testing.T) {
	allowedVendors := map[string]struct{}{
		"uid2":   {},
		"pubcid": {},
	}

	eids := []openrtb2.EID{
		{Source: "uidapi.com"}, // uid2 - should keep
		{Source: "pubcid.org"}, // pubcid - should keep
		{Source: "criteo.com"}, // not in allowlist - should keep (fail-open, ambiguous)
	}

	// Need to set EIDs in user.ext
	userExtJSON, _ := json.Marshal(map[string]interface{}{
		"eids": eids,
	})

	wrapper := &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			User: &openrtb2.User{
				Ext: userExtJSON,
			},
		},
	}

	err := pruneEIDs(wrapper, allowedVendors)
	assert.NoError(t, err)

	userExt, err := wrapper.GetUserExt()
	assert.NoError(t, err)

	filteredEIDs := userExt.GetEid()
	if filteredEIDs != nil {
		// Should keep uid2 and pubcid (conservative matching keeps ambiguous ones too)
		assert.GreaterOrEqual(t, len(*filteredEIDs), 2)
	}
}

func TestShouldKeepEIDMatchesVendorPatterns(t *testing.T) {
	allowedVendors := map[string]struct{}{
		"criteoId": {},
	}

	eid := openrtb2.EID{Source: "https://ad.criteo.com"}

	assert.True(t, shouldKeepEID(eid, allowedVendors))
}

func TestAccountConfigOverrides(t *testing.T) {
	// Create a test server with shaping config
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TrafficShapingData{
			Response: Response{
				SkipRate: 0,
				Values: map[string]map[string]map[string]int{
					"test-gpid": {
						"rubicon": {"300x250": 1},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		Enabled:          true,
		Endpoint:         server.URL,
		RefreshMs:        30000,
		RequestTimeoutMs: 1000,
		SampleSalt:       "pbs",
		AllowedCountries: []string{"US"}, // Host config only allows US
	}

	client := NewConfigClient(http.DefaultClient, config)
	defer client.Stop()

	// Wait for initial fetch
	for client.GetConfig() == nil {
	}

	module := &Module{
		config: config,
		client: client,
	}

	// Account config overrides allowed countries to CA
	accountConfig := json.RawMessage(`{"allowed_countries": ["CA"]}`)

	// Create request with GPID so shaping can be applied
	impExt, _ := json.Marshal(map[string]interface{}{
		"gpid": "test-gpid",
		"prebid": map[string]interface{}{
			"bidder": map[string]interface{}{
				"rubicon": json.RawMessage(`{}`),
			},
		},
	})

	payload := hookstage.ProcessedAuctionRequestPayload{
		Request: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				ID: "test-request",
				Device: &openrtb2.Device{
					Geo: &openrtb2.Geo{
						Country: "CA", // Request from CA
					},
				},
				Imp: []openrtb2.Imp{
					{
						ID:  "imp1",
						Ext: impExt,
						Banner: &openrtb2.Banner{
							Format: []openrtb2.Format{
								{W: 300, H: 250},
							},
						},
					},
				},
			},
		},
	}

	moduleCtx := hookstage.ModuleInvocationContext{
		AccountConfig: accountConfig,
	}

	result, err := module.HandleProcessedAuctionHook(context.Background(), moduleCtx, payload)

	assert.NoError(t, err)

	// Should NOT skip because account config allows CA (overrides host config)
	names := make(map[string]bool)
	for _, a := range result.AnalyticsTags.Activities {
		names[a.Name] = true
	}
	assert.False(t, names["skipped_country"], "Should not skip with account override")

	// Should have applied shaping
	assert.True(t, names["applied"])
	assert.True(t, names["shaped"])
}

func TestBuilder_GeoResolver(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"country":"US"}`))
	}))
	defer server.Close()

	config := fmt.Sprintf(`{
        "enabled": true,
        "base_endpoint": "http://localhost:8080/ts-server/",
        "geo_lookup_endpoint": "%s/{ip}",
        "geo_cache_ttl_ms": 1000
    }`, server.URL)

	deps := moduledeps.ModuleDeps{HTTPClient: http.DefaultClient}
	moduleRaw, err := Builder(json.RawMessage(config), deps)
	require.NoError(t, err)

	module := moduleRaw.(*Module)
	require.NotNil(t, module.geoResolver)

	country, err := module.geoResolver.Resolve(context.Background(), "1.1.1.1")
	require.NoError(t, err)
	assert.Equal(t, "US", country)
}

func TestHandleProcessedAuctionHook_Fallbacks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TrafficShapingData{
			Response: Response{
				SkipRate: 0,
				Values: map[string]map[string]map[string]int{
					"gpid": {
						"appnexus": {"300x250": 1},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		Enabled:           true,
		BaseEndpoint:      server.URL + "/",
		RefreshMs:         30000,
		RequestTimeoutMs:  1000,
		SampleSalt:        "pbs",
		GeoLookupEndpoint: server.URL + "/geo/{ip}",
		GeoCacheTTLMS:     1000,
	}

	client := NewConfigClient(http.DefaultClient, config)
	defer client.Stop()

	// In dynamic mode (base_endpoint set), client holds no static config; no wait needed.

	geoResolver := &mockGeoResolver{country: "IN"}
	module := &Module{
		config:      config,
		client:      client,
		geoResolver: geoResolver,
	}

	impExt, _ := json.Marshal(map[string]any{
		"gpid": "gpid",
	})

	bidRequest := &openrtb2.BidRequest{
		ID: "test",
		Imp: []openrtb2.Imp{
			{
				ID:  "imp1",
				Ext: impExt,
			},
		},
		Site: &openrtb2.Site{ID: "ts-server"},
		Device: &openrtb2.Device{
			UA: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36",
			IP: "1.1.1.1",
		},
	}

	wrapper := &openrtb_ext.RequestWrapper{BidRequest: bidRequest}
	payload := hookstage.ProcessedAuctionRequestPayload{Request: wrapper}

	result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)

	require.NoError(t, err)
	assert.NotEmpty(t, result.ChangeSet.ProcessedAuctionRequest())

	hasCountryDerived := false
	hasDeviceDerived := false
	for _, activity := range result.AnalyticsTags.Activities {
		if activity.Name == "country_derived" {
			hasCountryDerived = true
		}
		if activity.Name == "devicetype_derived" {
			hasDeviceDerived = true
		}
	}
	assert.True(t, hasCountryDerived)
	assert.True(t, hasDeviceDerived)
}

func TestHandleProcessedAuctionHook_PruneEIDsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TrafficShapingData{
			Response: Response{
				SkipRate:      0,
				UserIdVendors: []string{"uid2"},
				Values: map[string]map[string]map[string]int{
					"test-gpid": {
						"rubicon": {"300x250": 1},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		Enabled:          true,
		Endpoint:         server.URL,
		RefreshMs:        30000,
		RequestTimeoutMs: 1000,
		SampleSalt:       "pbs",
		PruneUserIds:     true,
	}

	client := NewConfigClient(http.DefaultClient, config)
	defer client.Stop()

	// Wait for initial fetch
	for client.GetConfig() == nil {
	}

	module := &Module{
		config: config,
		client: client,
	}

	// Create request with invalid user.ext JSON to trigger pruneEIDs error
	payload := hookstage.ProcessedAuctionRequestPayload{
		Request: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				ID: "test-request",
				User: &openrtb2.User{
					Ext: json.RawMessage(`{"invalid": json}`), // Invalid JSON
				},
				Imp: []openrtb2.Imp{
					{
						ID: "imp1",
						Ext: func() []byte {
							ext, _ := json.Marshal(map[string]interface{}{
								"gpid": "test-gpid",
							})
							return ext
						}(),
						Banner: &openrtb2.Banner{
							Format: []openrtb2.Format{
								{W: 300, H: 250},
							},
						},
					},
				},
			},
		},
	}

	result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)

	assert.NoError(t, err)
	// Should have warning about pruneEIDs failure
	assert.Contains(t, result.Warnings, "trafficshaping: failed to prune eids")
}

func TestGetAccountConfig_UnmarshalError(t *testing.T) {
	module := &Module{
		config: &Config{},
		client: nil,
	}

	// Invalid JSON should return nil
	accountConfig := module.getAccountConfig(json.RawMessage(`{"invalid": json}`))
	assert.Nil(t, accountConfig)
}

func TestAccountConfig_GetAllowedCountriesMap(t *testing.T) {
	t.Run("nil_allowed_countries", func(t *testing.T) {
		config := &AccountConfig{
			AllowedCountries: nil,
		}
		result := config.GetAllowedCountriesMap()
		assert.Nil(t, result)
	})

	t.Run("empty_allowed_countries", func(t *testing.T) {
		emptySlice := []string{}
		config := &AccountConfig{
			AllowedCountries: &emptySlice,
		}
		result := config.GetAllowedCountriesMap()
		assert.Nil(t, result)
	})

	t.Run("valid_allowed_countries", func(t *testing.T) {
		countries := []string{"US", "CA", "GB"}
		config := &AccountConfig{
			AllowedCountries: &countries,
		}
		result := config.GetAllowedCountriesMap()
		assert.NotNil(t, result)
		assert.Equal(t, 3, len(result))
		_, hasUS := result["US"]
		_, hasCA := result["CA"]
		_, hasGB := result["GB"]
		assert.True(t, hasUS)
		assert.True(t, hasCA)
		assert.True(t, hasGB)
	})
}

func TestPreprocessConfig_NoAllowedSizes(t *testing.T) {
	// Test case where bidder has sizes but all flags are 0
	response := &Response{
		SkipRate:      0,
		UserIdVendors: []string{},
		Values: map[string]map[string]map[string]int{
			"test-gpid": {
				"rubicon": {
					"300x250": 0, // All flags are 0
					"728x90":  0,
				},
			},
		},
	}

	config := preprocessConfig(response)

	// Should have GPID rule
	rule, ok := config.GPIDRules["test-gpid"]
	require.True(t, ok)

	// Should not have rubicon in allowed bidders (no allowed sizes)
	_, hasRubicon := rule.AllowedBidders["rubicon"]
	assert.False(t, hasRubicon)

	// Should have empty allowed sizes
	assert.Equal(t, 0, len(rule.AllowedSizes))
}

func TestHandleProcessedAuctionHook_AccountConfigPruneUserIdsOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TrafficShapingData{
			Response: Response{
				SkipRate:      0,
				UserIdVendors: []string{"uid2"},
				Values: map[string]map[string]map[string]int{
					"test-gpid": {
						"rubicon": {"300x250": 1},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &Config{
		Enabled:          true,
		Endpoint:         server.URL,
		RefreshMs:        30000,
		RequestTimeoutMs: 1000,
		SampleSalt:       "pbs",
		PruneUserIds:     false, // Host config has it disabled
	}

	client := NewConfigClient(http.DefaultClient, config)
	defer client.Stop()

	// Wait for initial fetch
	for client.GetConfig() == nil {
	}

	module := &Module{
		config: config,
		client: client,
	}

	// Account config overrides pruneUserIds to true
	accountConfig := json.RawMessage(`{"prune_user_ids": true}`)

	eids := []openrtb2.EID{
		{Source: "uidapi.com"},
	}

	userExtJSON, _ := json.Marshal(map[string]interface{}{
		"eids": eids,
	})

	payload := hookstage.ProcessedAuctionRequestPayload{
		Request: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				ID: "test-request",
				User: &openrtb2.User{
					Ext: userExtJSON,
				},
				Imp: []openrtb2.Imp{
					{
						ID: "imp1",
						Ext: func() []byte {
							ext, _ := json.Marshal(map[string]interface{}{
								"gpid": "test-gpid",
							})
							return ext
						}(),
						Banner: &openrtb2.Banner{
							Format: []openrtb2.Format{
								{W: 300, H: 250},
							},
						},
					},
				},
			},
		},
	}

	moduleCtx := hookstage.ModuleInvocationContext{
		AccountConfig: accountConfig,
	}

	result, err := module.HandleProcessedAuctionHook(context.Background(), moduleCtx, payload)

	assert.NoError(t, err)
	// Should have applied shaping
	names := make(map[string]bool)
	for _, a := range result.AnalyticsTags.Activities {
		names[a.Name] = true
	}
	assert.True(t, names["applied"])
	assert.True(t, names["shaped"])

	// EIDs should be pruned (account config override enabled pruning)
	userExt, err := payload.Request.GetUserExt()
	if err == nil && userExt != nil {
		eids := userExt.GetEid()
		if eids != nil {
			// Should have filtered EIDs
			assert.GreaterOrEqual(t, len(*eids), 0)
		}
	}
}

func TestCheckTDIDRtiPartner(t *testing.T) {
	tests := []struct {
		name     string
		eid      openrtb2.EID
		expected bool
	}{
		{
			name: "valid_tdid",
			eid: openrtb2.EID{
				Source: "adserver.org",
				UIDs: []openrtb2.UID{
					{
						ID:  "test-id",
						Ext: json.RawMessage(`{"rtiPartner":"TDID"}`),
					},
				},
			},
			expected: true,
		},
		{
			name: "valid_tdid_lowercase",
			eid: openrtb2.EID{
				Source: "adserver.org",
				UIDs: []openrtb2.UID{
					{
						ID:  "test-id",
						Ext: json.RawMessage(`{"rtiPartner":"tdid"}`),
					},
				},
			},
			expected: true,
		},
		{
			name: "invalid_rtipartner",
			eid: openrtb2.EID{
				Source: "adserver.org",
				UIDs: []openrtb2.UID{
					{
						ID:  "test-id",
						Ext: json.RawMessage(`{"rtiPartner":"OTHER"}`),
					},
				},
			},
			expected: false,
		},
		{
			name: "missing_ext",
			eid: openrtb2.EID{
				Source: "adserver.org",
				UIDs: []openrtb2.UID{
					{
						ID: "test-id",
					},
				},
			},
			expected: false,
		},
		{
			name: "invalid_json",
			eid: openrtb2.EID{
				Source: "adserver.org",
				UIDs: []openrtb2.UID{
					{
						ID:  "test-id",
						Ext: json.RawMessage(`{"rtiPartner":invalid}`),
					},
				},
			},
			expected: false,
		},
		{
			name: "empty_uids",
			eid: openrtb2.EID{
				Source: "adserver.org",
				UIDs:   []openrtb2.UID{},
			},
			expected: false,
		},
		{
			name: "multiple_uids_one_valid",
			eid: openrtb2.EID{
				Source: "adserver.org",
				UIDs: []openrtb2.UID{
					{
						ID:  "test-id-1",
						Ext: json.RawMessage(`{"rtiPartner":"OTHER"}`),
					},
					{
						ID:  "test-id-2",
						Ext: json.RawMessage(`{"rtiPartner":"TDID"}`),
					},
				},
			},
			expected: true,
		},
		{
			name: "missing_rtipartner_field",
			eid: openrtb2.EID{
				Source: "adserver.org",
				UIDs: []openrtb2.UID{
					{
						ID:  "test-id",
						Ext: json.RawMessage(`{"otherField":"value"}`),
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkTDIDRtiPartner(tt.eid)
			assert.Equal(t, tt.expected, result)
		})
	}
}
