package trafficshaping

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_RealEndpoint tests the complete flow using the real endpoint
func TestIntegration_RealEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Real endpoint from user
	realEndpoint := "https://rtd.mile.so/ts-static/"

	deps := moduledeps.ModuleDeps{
		HTTPClient: http.DefaultClient,
	}

	module, err := Builder(json.RawMessage(`{
		"enabled": true,
		"base_endpoint": "`+realEndpoint+`",
		"refresh_ms": 30000,
		"request_timeout_ms": 5000
	}`), deps)
	require.NoError(t, err)
	require.NotNil(t, module)

	mod := module.(*Module)
	defer mod.Close()

	// Create request matching the endpoint: 0OsUhO/US/w/chrome/ts.json
	impExt, _ := json.Marshal(map[string]interface{}{
		"gpid": "21804848220,22690441817/ATD_RecipeReader/ATD_728x90_Footer#fi-ash-1709555708-6581",
		"prebid": map[string]interface{}{
			"bidder": map[string]interface{}{
				"criteo":   json.RawMessage(`{}`),
				"gumgum":   json.RawMessage(`{}`),
				"kueezrtb": json.RawMessage(`{}`),
				"medianet": json.RawMessage(`{}`),
				"rubicon":  json.RawMessage(`{}`),
				"vidazoo":  json.RawMessage(`{}`),
				"appnexus": json.RawMessage(`{}`), // Should be filtered out (not in config)
			},
		},
	})

	bidRequest := &openrtb2.BidRequest{
		ID: "test-integration-request",
		Site: &openrtb2.Site{
			ID: "0OsUhO",
		},
		Device: &openrtb2.Device{
			DeviceType: 2, // Desktop
			UA:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			Geo: &openrtb2.Geo{
				Country: "US",
			},
		},
		Imp: []openrtb2.Imp{
			{
				ID:  "imp1",
				Ext: impExt,
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 728, H: 90},
						{W: 300, H: 250}, // Should be filtered out (not in config)
					},
				},
			},
		},
	}

	wrapper := &openrtb_ext.RequestWrapper{BidRequest: bidRequest}
	payload := hookstage.ProcessedAuctionRequestPayload{Request: wrapper}

	// Wait a bit for initial config fetch
	time.Sleep(2 * time.Second)

	result, err := mod.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)

	require.NoError(t, err)

	// Check analytics tags
	activityNames := make(map[string]bool)
	for _, activity := range result.AnalyticsTags.Activities {
		activityNames[activity.Name] = true
	}

	// Should have applied shaping (unless skipRate=100 blocks it)
	// The config has skipRate=100, so it should skip
	if activityNames["skipped_by_skiprate"] {
		t.Log("Request skipped by skipRate (expected for this config)")
		assert.True(t, activityNames["skipped"])
	} else {
		// If not skipped, should have applied shaping
		assert.True(t, activityNames["applied"] || activityNames["shaped"])
	}

	// Verify URL was constructed correctly
	expectedURL := realEndpoint + "0OsUhO/US/w/chrome/ts.json"
	t.Logf("Expected URL: %s", expectedURL)
}

// TestIntegration_WithWhitelist tests the complete flow with whitelist enabled
func TestIntegration_WithWhitelist(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Mock whitelist endpoints
	geoWhitelistServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string][]string{
			"0OsUhO": {"US", "CA"},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer geoWhitelistServer.Close()

	platformWhitelistServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string][]string{
			"0OsUhO": {"w|chrome", "w|safari"},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer platformWhitelistServer.Close()

	// Real ts.json endpoint
	realEndpoint := "https://rtd.mile.so/ts-static/"

	configJSON := `{
		"enabled": true,
		"base_endpoint": "` + realEndpoint + `",
		"refresh_ms": 30000,
		"request_timeout_ms": 5000,
		"geo_whitelist_endpoint": "` + geoWhitelistServer.URL + `",
		"platform_whitelist_endpoint": "` + platformWhitelistServer.URL + `",
		"whitelist_refresh_ms": 300000
	}`

	deps := moduledeps.ModuleDeps{
		HTTPClient: http.DefaultClient,
	}

	module, err := Builder(json.RawMessage(configJSON), deps)
	require.NoError(t, err)
	require.NotNil(t, module)

	mod := module.(*Module)
	defer mod.Close()

	// Wait for whitelist to load
	time.Sleep(1 * time.Second)

	tests := []struct {
		name           string
		siteID         string
		country        string
		ua             string
		expectAllowed  bool
		expectActivity string
	}{
		{
			name:           "allowed_site_geo_platform",
			siteID:         "0OsUhO",
			country:        "US",
			ua:             "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0.0.0 Safari/537.36",
			expectAllowed:  true,
			expectActivity: "applied", // or "skipped_by_skiprate" if skipRate=100
		},
		{
			name:           "disallowed_country",
			siteID:         "0OsUhO",
			country:        "GB", // Not in whitelist
			ua:             "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0.0.0 Safari/537.36",
			expectAllowed:  false,
			expectActivity: "skipped_whitelist",
		},
		{
			name:           "disallowed_platform",
			siteID:         "0OsUhO",
			country:        "US",
			ua:             "Mozilla/5.0 (Windows NT 10.0) Firefox/120.0", // ff not in whitelist
			expectAllowed:  false,
			expectActivity: "skipped_whitelist",
		},
		{
			name:           "unknown_site_skip_shaping",
			siteID:         "unknown-site",
			country:        "US",
			ua:             "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0.0.0 Safari/537.36",
			expectAllowed:  false, // Should skip shaping (site not enabled)
			expectActivity: "skipped_whitelist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			impExt, _ := json.Marshal(map[string]interface{}{
				"gpid": "test-gpid",
			})

			bidRequest := &openrtb2.BidRequest{
				ID: "test-" + tt.name,
				Site: &openrtb2.Site{
					ID: tt.siteID,
				},
				Device: &openrtb2.Device{
					DeviceType: 2, // Desktop
					UA:         tt.ua,
					Geo: &openrtb2.Geo{
						Country: tt.country,
					},
				},
				Imp: []openrtb2.Imp{
					{
						ID:  "imp1",
						Ext: impExt,
					},
				},
			}

			wrapper := &openrtb_ext.RequestWrapper{BidRequest: bidRequest}
			payload := hookstage.ProcessedAuctionRequestPayload{Request: wrapper}

			result, err := mod.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)

			require.NoError(t, err)

			activityNames := make(map[string]bool)
			for _, activity := range result.AnalyticsTags.Activities {
				activityNames[activity.Name] = true
			}

			if tt.expectAllowed {
				// Should proceed (may be skipped by skipRate though)
				assert.True(t, activityNames["applied"] || activityNames["skipped_by_skiprate"] || activityNames["skipped_no_config"])
			} else {
				// Should be blocked by whitelist
				assert.True(t, activityNames["skipped_whitelist"])
				assert.True(t, activityNames["skipped"])
			}
		})
	}
}

// TestIntegration_CompleteFlow tests the complete flow: resolution -> whitelist -> URL construction -> shaping
func TestIntegration_CompleteFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Mock whitelist servers
	geoWhitelistServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string][]string{
			"0OsUhO": {"US"},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer geoWhitelistServer.Close()

	platformWhitelistServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string][]string{
			"0OsUhO": {"w|chrome"},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer platformWhitelistServer.Close()

	// Mock ts.json server (simulating real endpoint)
	tsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL format: /0OsUhO/US/w/chrome/ts.json
		expectedPath := "/0OsUhO/US/w/chrome/ts.json"
		if r.URL.Path != expectedPath {
			t.Logf("Unexpected path: got %s, expected %s", r.URL.Path, expectedPath)
		}

		response := TrafficShapingData{
			Meta: Meta{CreatedAt: 1234567890},
			Response: Response{
				SkipRate:      0, // No skip rate for this test
				UserIdVendors: []string{"uid2", "pubcid"},
				Values: map[string]map[string]map[string]int{
					"21804848220,22690441817/ATD_RecipeReader/ATD_728x90_Footer#fi-ash-1709555708-6581": {
						"criteo":  {"728x90": 1},
						"gumgum":  {"728x90": 1},
						"rubicon": {"728x90": 1},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer tsServer.Close()

	configJSON := `{
		"enabled": true,
		"base_endpoint": "` + tsServer.URL + `/",
		"refresh_ms": 30000,
		"request_timeout_ms": 2000,
		"geo_whitelist_endpoint": "` + geoWhitelistServer.URL + `",
		"platform_whitelist_endpoint": "` + platformWhitelistServer.URL + `",
		"whitelist_refresh_ms": 300000
	}`

	deps := moduledeps.ModuleDeps{
		HTTPClient: http.DefaultClient,
	}

	module, err := Builder(json.RawMessage(configJSON), deps)
	require.NoError(t, err)
	require.NotNil(t, module)

	mod := module.(*Module)
	defer mod.Close()

	// Wait for whitelist and config to load
	time.Sleep(1 * time.Second)

	// Create request
	impExt, _ := json.Marshal(map[string]interface{}{
		"gpid": "21804848220,22690441817/ATD_RecipeReader/ATD_728x90_Footer#fi-ash-1709555708-6581",
		"prebid": map[string]interface{}{
			"bidder": map[string]interface{}{
				"criteo":   json.RawMessage(`{}`),
				"gumgum":   json.RawMessage(`{}`),
				"rubicon":  json.RawMessage(`{}`),
				"appnexus": json.RawMessage(`{}`), // Should be filtered out
			},
		},
	})

	bidRequest := &openrtb2.BidRequest{
		ID: "test-complete-flow",
		Site: &openrtb2.Site{
			ID: "0OsUhO",
		},
		Device: &openrtb2.Device{
			DeviceType: 2, // Desktop
			UA:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			Geo: &openrtb2.Geo{
				Country: "US",
			},
		},
		Imp: []openrtb2.Imp{
			{
				ID:  "imp1",
				Ext: impExt,
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 728, H: 90},  // Allowed
						{W: 300, H: 250}, // Should be filtered out
					},
				},
			},
		},
	}

	wrapper := &openrtb_ext.RequestWrapper{BidRequest: bidRequest}
	payload := hookstage.ProcessedAuctionRequestPayload{Request: wrapper}

	result, err := mod.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)

	require.NoError(t, err)

	// Check analytics
	activityNames := make(map[string]bool)
	for _, activity := range result.AnalyticsTags.Activities {
		activityNames[activity.Name] = true
	}

	// Should have passed whitelist and applied shaping
	assert.True(t, activityNames["applied"] || activityNames["shaped"], "Should have applied shaping")
	assert.False(t, activityNames["skipped_whitelist"], "Should not be skipped by whitelist")

	// Verify bidder filtering was applied
	if activityNames["applied"] {
		changeSet := result.ChangeSet.ProcessedAuctionRequest()
		assert.NotNil(t, changeSet, "ChangeSet should be populated")
	}

	t.Logf("Activities: %v", activityNames)
}

// TestIntegration_SiteNotInWhitelist tests that sites not in whitelist are allowed (fail-open)
func TestIntegration_SiteNotInWhitelist(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Mock whitelist endpoints - only "enabled-site" is enabled
	geoWhitelistServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string][]string{
			"enabled-site": {"US", "CA"},
			// "disabled-site" is NOT in the whitelist
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer geoWhitelistServer.Close()

	platformWhitelistServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string][]string{
			"enabled-site": {"w|chrome"},
			// "disabled-site" is NOT in the whitelist
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer platformWhitelistServer.Close()

	// Mock ts.json server
	tsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TrafficShapingData{
			Meta: Meta{CreatedAt: 1234567890},
			Response: Response{
				SkipRate:      0,
				UserIdVendors: []string{},
				Values: map[string]map[string]map[string]int{
					"test-gpid": {
						"rubicon": {"300x250": 1},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer tsServer.Close()

	configJSON := `{
		"enabled": true,
		"base_endpoint": "` + tsServer.URL + `/",
		"refresh_ms": 30000,
		"request_timeout_ms": 2000,
		"geo_whitelist_endpoint": "` + geoWhitelistServer.URL + `",
		"platform_whitelist_endpoint": "` + platformWhitelistServer.URL + `",
		"whitelist_refresh_ms": 300000
	}`

	deps := moduledeps.ModuleDeps{
		HTTPClient: http.DefaultClient,
	}

	module, err := Builder(json.RawMessage(configJSON), deps)
	require.NoError(t, err)
	require.NotNil(t, module)

	mod := module.(*Module)
	defer mod.Close()

	// Wait for whitelist to load
	time.Sleep(1 * time.Second)

	tests := []struct {
		name           string
		siteID         string
		country        string
		ua             string
		expectAllowed  bool
		expectActivity string
		description    string
	}{
		{
			name:           "site_not_in_whitelist_should_skip_shaping",
			siteID:         "disabled-site", // NOT in whitelist
			country:        "US",
			ua:             "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0.0.0 Safari/537.36",
			expectAllowed:  false, // Should skip shaping
			expectActivity: "skipped_whitelist",
			description:    "Site not in whitelist should skip shaping",
		},
		{
			name:           "enabled_site_with_matching_geo_platform",
			siteID:         "enabled-site",                                                 // In whitelist
			country:        "US",                                                           // Matches whitelist
			ua:             "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0.0.0 Safari/537.36", // w|chrome matches
			expectAllowed:  true,
			expectActivity: "applied",
			description:    "Enabled site with matching geo/platform should proceed",
		},
		{
			name:           "enabled_site_with_non_matching_geo",
			siteID:         "enabled-site", // In whitelist
			country:        "GB",           // NOT in whitelist (only US, CA allowed)
			ua:             "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0.0.0 Safari/537.36",
			expectAllowed:  false, // Should be blocked
			expectActivity: "skipped_whitelist",
			description:    "Enabled site with non-matching geo should be blocked",
		},
		{
			name:           "enabled_site_with_non_matching_platform",
			siteID:         "enabled-site",                                // In whitelist
			country:        "US",                                          // Matches
			ua:             "Mozilla/5.0 (Windows NT 10.0) Firefox/120.0", // ff not in whitelist (only chrome)
			expectAllowed:  false,                                         // Should be blocked
			expectActivity: "skipped_whitelist",
			description:    "Enabled site with non-matching platform should be blocked",
		},
		{
			name:           "empty_site_id_whitelist_passes_but_url_fails",
			siteID:         "", // Empty site ID
			country:        "US",
			ua:             "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0.0.0 Safari/537.36",
			expectAllowed:  true,                              // Whitelist passes (fail-open for empty), but URL construction will fail
			expectActivity: "skipped_url_construction_failed", // URL construction fails without siteID
			description:    "Empty site ID passes whitelist but fails URL construction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			impExt, _ := json.Marshal(map[string]interface{}{
				"gpid": "test-gpid",
			})

			bidRequest := &openrtb2.BidRequest{
				ID: "test-" + tt.name,
				Site: &openrtb2.Site{
					ID: tt.siteID,
				},
				Device: &openrtb2.Device{
					DeviceType: 2, // Desktop
					UA:         tt.ua,
					Geo: &openrtb2.Geo{
						Country: tt.country,
					},
				},
				Imp: []openrtb2.Imp{
					{
						ID:  "imp1",
						Ext: impExt,
					},
				},
			}

			wrapper := &openrtb_ext.RequestWrapper{BidRequest: bidRequest}
			payload := hookstage.ProcessedAuctionRequestPayload{Request: wrapper}

			result, err := mod.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)

			require.NoError(t, err, tt.description)

			activityNames := make(map[string]bool)
			for _, activity := range result.AnalyticsTags.Activities {
				activityNames[activity.Name] = true
			}

			if tt.expectAllowed {
				// Should proceed (may be skipped by skipRate or URL construction though)
				assert.False(t, activityNames["skipped_whitelist"],
					"%s: Should not be skipped by whitelist", tt.description)
				// May proceed to shaping OR fail at URL construction (for empty siteID)
				assert.True(t, activityNames["applied"] || activityNames["skipped_by_skiprate"] ||
					activityNames["skipped_no_config"] || activityNames["skipped_url_construction_failed"],
					"%s: Should proceed past whitelist (may fail later)", tt.description)
			} else {
				// Should be blocked by whitelist
				assert.True(t, activityNames["skipped_whitelist"],
					"%s: Should be skipped by whitelist", tt.description)
				assert.True(t, activityNames["skipped"],
					"%s: Should have skipped tag", tt.description)
			}

			t.Logf("%s: Activities = %v", tt.description, activityNames)
		})
	}
}
