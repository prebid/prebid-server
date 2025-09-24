package scope3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/coocood/freecache"
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestModuleDeps(t *testing.T) moduledeps.ModuleDeps {
	t.Helper()
	return moduledeps.ModuleDeps{
		HTTPClient: http.DefaultClient,
	}
}

func getTestEntrypointPayload(t *testing.T) hookstage.EntrypointPayload {
	body := []byte(`{}`)
	return hookstage.EntrypointPayload{
		Request: httptest.NewRequest(http.MethodPost, "/openrtb2/auction", bytes.NewBuffer(body)),
		Body:    body,
	}
}

func TestBuilder(t *testing.T) {
	config := json.RawMessage(`{
		"enabled": true,
		"endpoint": "https://rtdp.scope3.com/amazonaps/rtii",
		"auth_key": "test-key",
		"timeout_ms": 1000,
		"cache_ttl_seconds": 60,
		"add_to_targeting": false
	}`)

	deps := moduledeps.ModuleDeps{HTTPClient: http.DefaultClient}
	module, err := Builder(config, deps)

	assert.NoError(t, err)
	assert.NotNil(t, module)
	assert.IsType(t, &Module{}, module)

	m := module.(*Module)
	assert.Equal(t, "https://rtdp.scope3.com/amazonaps/rtii", m.cfg.Endpoint)
	assert.Equal(t, "test-key", m.cfg.AuthKey)
	assert.Equal(t, 1000, m.cfg.Timeout)
	assert.Equal(t, 60, m.cfg.CacheTTL)
	assert.Equal(t, false, m.cfg.AddToTargeting)
	assert.NotNil(t, m.cache)
}

func TestBuilderInvalidConfig(t *testing.T) {
	config := json.RawMessage(`invalid json`)
	deps := moduledeps.ModuleDeps{HTTPClient: http.DefaultClient}

	module, err := Builder(config, deps)

	assert.Error(t, err)
	assert.Nil(t, module)
}

func TestHandleEntrypointHook(t *testing.T) {
	module := &Module{
		asyncRequestPool: &sync.Pool{
			New: func() any {
				return &AsyncRequest{}
			},
		}}
	ctx := context.Background()
	miCtx := hookstage.ModuleInvocationContext{}
	payload := getTestEntrypointPayload(t)

	result, err := module.HandleEntrypointHook(ctx, miCtx, payload)

	assert.NoError(t, err)
	assert.NotNil(t, result.ModuleContext[asyncRequestKey])
}

func TestHandleAuctionResponseHook_NoSegments(t *testing.T) {
	module := &Module{
		asyncRequestPool: &sync.Pool{
			New: func() any {
				return &AsyncRequest{}
			},
		},
	}
	ctx := context.Background()
	miCtx := hookstage.ModuleInvocationContext{
		ModuleContext: hookstage.ModuleContext{
			"segments": &sync.Map{},
		},
	}
	payload := hookstage.AuctionResponsePayload{}

	result, err := module.HandleAuctionResponseHook(ctx, miCtx, payload)

	assert.NoError(t, err)
	assert.Empty(t, result.ChangeSet)
}

func TestBuilderDefaults(t *testing.T) {
	config := json.RawMessage(`{
		"enabled": true,
		"auth_key": "test-key"
	}`)

	deps := moduledeps.ModuleDeps{HTTPClient: http.DefaultClient}
	module, err := Builder(config, deps)

	assert.NoError(t, err)
	m := module.(*Module)
	assert.Equal(t, "https://rtdp.scope3.com/prebid/rtii", m.cfg.Endpoint)
	assert.Equal(t, 1000, m.cfg.Timeout)
	assert.Equal(t, 60, m.cfg.CacheTTL)
	assert.Equal(t, false, m.cfg.AddToTargeting)
}

func TestScope3APIIntegration(t *testing.T) {
	// Create mock Scope3 API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-auth-key", r.Header.Get("x-scope3-auth"))

		// Return mock Scope3 response with segments
		response := `{
			"data": [
				{
					"destination": "triplelift.com",
					"imp": [
						{
							"id": "test-imp-1",
							"ext": {
								"scope3": {
									"segments": [
										{"id": "gmp_eligible"},
										{"id": "gmp_plus_eligible"}
									]
								}
							}
						}
					]
				}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
	defer mockServer.Close()

	// Create module with mock server endpoint
	config := json.RawMessage(`{
		"endpoint": "` + mockServer.URL + `",
		"auth_key": "test-auth-key",
		"timeout_ms": 1000,
		"cache_ttl_seconds": 60,
		"add_to_targeting": false
	}`)

	moduleInterface, err := Builder(config, getTestModuleDeps(t))
	require.NoError(t, err)
	module := moduleInterface.(*Module)

	// Create test bid request
	width := int64(300)
	height := int64(250)
	bidRequest := &openrtb2.BidRequest{
		ID: "test-auction",
		Imp: []openrtb2.Imp{{
			ID:     "test-imp-1",
			Banner: &openrtb2.Banner{W: &width, H: &height},
		}},
		Site: &openrtb2.Site{
			Domain: "example.com",
			Page:   "https://example.com/test-page",
		},
		User: &openrtb2.User{
			ID: "test-user",
			Ext: json.RawMessage(`{
				"eids": [
					{
						"source": "liveramp.com",
						"uids": [{"id": "test-ramp-id"}]
					}
				]
			}`),
		},
	}

	// Test fetchScope3Segments
	ctx := context.Background()
	segments, err := module.fetchScope3Segments(ctx, bidRequest)
	require.NoError(t, err)
	assert.Len(t, segments, 2)
	assert.Contains(t, segments, "gmp_eligible")
	assert.Contains(t, segments, "gmp_plus_eligible")
	assert.NotContains(t, segments, "triplelift.com") // Should not include destination
}

func TestScope3APIIntegrationWithTargeting(t *testing.T) {
	// Create mock server that returns segments
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"data": [
				{
					"destination": "triplelift.com",
					"imp": [
						{
							"id": "test-imp-1",
							"ext": {
								"scope3": {
									"segments": [
										{"id": "test_segment_1"},
										{"id": "test_segment_2"}
									]
								}
							}
						}
					]
				}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
	defer mockServer.Close()

	// Create module with targeting enabled
	config := json.RawMessage(`{
		"endpoint": "` + mockServer.URL + `",
		"auth_key": "test-auth-key",
		"timeout_ms": 1000,
		"add_to_targeting": true
	}`)

	moduleInterface, err := Builder(config, getTestModuleDeps(t))
	require.NoError(t, err)
	module := moduleInterface.(*Module)

	// Test full hook workflow
	ctx := context.Background()

	// Test entrypoint hook
	entrypointResult, err := module.HandleEntrypointHook(ctx, hookstage.ModuleInvocationContext{}, getTestEntrypointPayload(t))
	require.NoError(t, err)
	assert.NotNil(t, entrypointResult.ModuleContext[asyncRequestKey])

	// Create test request payload
	width := int64(300)
	height := int64(250)
	bidRequest := openrtb2.BidRequest{
		ID: "test-auction",
		Imp: []openrtb2.Imp{{
			ID:     "test-imp-1",
			Banner: &openrtb2.Banner{W: &width, H: &height},
		}},
		Site: &openrtb2.Site{
			Domain: "example.com",
			Page:   "https://example.com/test",
		},
	}
	requestPayload, _ := json.Marshal(bidRequest)

	// Test raw auction hook
	miCtx := hookstage.ModuleInvocationContext{
		ModuleContext: entrypointResult.ModuleContext,
	}
	_, err = module.HandleRawAuctionHook(ctx, miCtx, requestPayload)
	require.NoError(t, err)

	// Test auction response hook
	responsePayload := hookstage.AuctionResponsePayload{
		BidResponse: &openrtb2.BidResponse{
			ID:  "test-response",
			Ext: json.RawMessage(`{}`),
		},
	}

	responseResult, err := module.HandleAuctionResponseHook(ctx, miCtx, responsePayload)
	require.NoError(t, err)

	// Verify the response was modified
	assert.True(t, len(responseResult.ChangeSet.Mutations()) > 0)

	// Apply the mutations and check the result
	modifiedPayload := responsePayload
	for _, mutation := range responseResult.ChangeSet.Mutations() {
		var err error
		modifiedPayload, err = mutation.Apply(modifiedPayload)
		require.NoError(t, err)
	}

	// Parse the modified response
	var extMap map[string]interface{}
	err = json.Unmarshal(modifiedPayload.BidResponse.Ext, &extMap)
	require.NoError(t, err)

	// Verify scope3 section exists
	scope3Data, exists := extMap["scope3"].(map[string]interface{})
	require.True(t, exists)
	segments, exists := scope3Data["segments"].([]interface{})
	require.True(t, exists)
	assert.Len(t, segments, 2)

	// Verify targeting section exists (add_to_targeting: true)
	prebidData, exists := extMap["prebid"].(map[string]interface{})
	require.True(t, exists)
	targetingData, exists := prebidData["targeting"].(map[string]interface{})
	require.True(t, exists)

	// Check individual targeting keys
	assert.Equal(t, "true", targetingData["test_segment_1"])
	assert.Equal(t, "true", targetingData["test_segment_2"])
}

func TestScope3APIIntegrationWithExistingPrebidTargeting(t *testing.T) {
	// Create mock server that returns segments
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"data": [
				{
					"destination": "triplelift.com",
					"imp": [
						{
							"id": "test-imp-1",
							"ext": {
								"scope3": {
									"segments": [
										{"id": "test_segment_1"},
										{"id": "test_segment_2"}
									]
								}
							}
						}
					]
				}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
	defer mockServer.Close()

	// Create module with targeting enabled
	config := json.RawMessage(`{
		"endpoint": "` + mockServer.URL + `",
		"auth_key": "test-auth-key",
		"timeout_ms": 1000,
		"add_to_targeting": true
	}`)

	moduleInterface, err := Builder(config, getTestModuleDeps(t))
	require.NoError(t, err)
	module := moduleInterface.(*Module)

	// Test full hook workflow
	ctx := context.Background()

	// Test entrypoint hook
	entrypointResult, err := module.HandleEntrypointHook(ctx, hookstage.ModuleInvocationContext{}, getTestEntrypointPayload(t))
	require.NoError(t, err)
	assert.NotNil(t, entrypointResult.ModuleContext[asyncRequestKey])

	// Create test request payload
	width := int64(300)
	height := int64(250)
	bidRequest := openrtb2.BidRequest{
		ID: "test-auction",
		Imp: []openrtb2.Imp{{
			ID:     "test-imp-1",
			Banner: &openrtb2.Banner{W: &width, H: &height},
		}},
		Site: &openrtb2.Site{
			Domain: "example.com",
			Page:   "https://example.com/test",
		},
	}
	requestPayload, _ := json.Marshal(bidRequest)

	// Test raw auction hook
	miCtx := hookstage.ModuleInvocationContext{
		ModuleContext: entrypointResult.ModuleContext,
	}
	_, err = module.HandleRawAuctionHook(ctx, miCtx, requestPayload)
	require.NoError(t, err)

	// Test auction response hook
	responsePayload := hookstage.AuctionResponsePayload{
		BidResponse: &openrtb2.BidResponse{
			ID:  "test-response",
			Ext: json.RawMessage(`{"prebid":{"targeting":{"segment_existing":"true"}}}`),
		},
	}

	responseResult, err := module.HandleAuctionResponseHook(ctx, miCtx, responsePayload)
	require.NoError(t, err)

	// Verify the response was modified
	assert.True(t, len(responseResult.ChangeSet.Mutations()) > 0)

	// Apply the mutations and check the result
	modifiedPayload := responsePayload
	for _, mutation := range responseResult.ChangeSet.Mutations() {
		var err error
		modifiedPayload, err = mutation.Apply(modifiedPayload)
		require.NoError(t, err)
	}

	// Parse the modified response
	var extMap map[string]interface{}
	err = json.Unmarshal(modifiedPayload.BidResponse.Ext, &extMap)
	require.NoError(t, err)

	// Verify scope3 section exists
	scope3Data, exists := extMap["scope3"].(map[string]interface{})
	require.True(t, exists)
	segments, exists := scope3Data["segments"].([]interface{})
	require.True(t, exists)
	assert.Len(t, segments, 2)

	// Verify targeting section exists (add_to_targeting: true)
	prebidData, exists := extMap["prebid"].(map[string]interface{})
	require.True(t, exists)
	targetingData, exists := prebidData["targeting"].(map[string]interface{})
	require.True(t, exists)

	// Check individual targeting keys
	assert.Equal(t, "true", targetingData["segment_existing"])
	assert.Equal(t, "true", targetingData["test_segment_1"])
	assert.Equal(t, "true", targetingData["test_segment_2"])
}

func TestScope3APIIntegrationWithExistingPrebidNoTargeting(t *testing.T) {
	// Create mock server that returns segments
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"data": [
				{
					"destination": "triplelift.com",
					"imp": [
						{
							"id": "test-imp-1",
							"ext": {
								"scope3": {
									"segments": [
										{"id": "test_segment_1"},
										{"id": "test_segment_2"}
									]
								}
							}
						}
					]
				}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
	defer mockServer.Close()

	// Create module with targeting enabled
	config := json.RawMessage(`{
		"endpoint": "` + mockServer.URL + `",
		"auth_key": "test-auth-key",
		"timeout_ms": 1000,
		"add_to_targeting": true
	}`)

	moduleInterface, err := Builder(config, getTestModuleDeps(t))
	require.NoError(t, err)
	module := moduleInterface.(*Module)

	// Test full hook workflow
	ctx := context.Background()

	// Test entrypoint hook
	entrypointResult, err := module.HandleEntrypointHook(ctx, hookstage.ModuleInvocationContext{}, getTestEntrypointPayload(t))
	require.NoError(t, err)
	assert.NotNil(t, entrypointResult.ModuleContext[asyncRequestKey])

	// Create test request payload
	width := int64(300)
	height := int64(250)
	bidRequest := openrtb2.BidRequest{
		ID: "test-auction",
		Imp: []openrtb2.Imp{{
			ID:     "test-imp-1",
			Banner: &openrtb2.Banner{W: &width, H: &height},
		}},
		Site: &openrtb2.Site{
			Domain: "example.com",
			Page:   "https://example.com/test",
		},
	}
	requestPayload, _ := json.Marshal(bidRequest)

	// Test raw auction hook
	miCtx := hookstage.ModuleInvocationContext{
		ModuleContext: entrypointResult.ModuleContext,
	}
	_, err = module.HandleRawAuctionHook(ctx, miCtx, requestPayload)
	require.NoError(t, err)

	// Test auction response hook
	responsePayload := hookstage.AuctionResponsePayload{
		BidResponse: &openrtb2.BidResponse{
			ID:  "test-response",
			Ext: json.RawMessage(`{"prebid":{"something_else":"true"}}`),
		},
	}

	responseResult, err := module.HandleAuctionResponseHook(ctx, miCtx, responsePayload)
	require.NoError(t, err)

	// Verify the response was modified
	assert.True(t, len(responseResult.ChangeSet.Mutations()) > 0)

	// Apply the mutations and check the result
	modifiedPayload := responsePayload
	for _, mutation := range responseResult.ChangeSet.Mutations() {
		var err error
		modifiedPayload, err = mutation.Apply(modifiedPayload)
		require.NoError(t, err)
	}

	// Parse the modified response
	var extMap map[string]interface{}
	err = json.Unmarshal(modifiedPayload.BidResponse.Ext, &extMap)
	require.NoError(t, err)

	// Verify scope3 section exists
	scope3Data, exists := extMap["scope3"].(map[string]interface{})
	require.True(t, exists)
	segments, exists := scope3Data["segments"].([]interface{})
	require.True(t, exists)
	assert.Len(t, segments, 2)

	// Verify targeting section exists (add_to_targeting: true)
	prebidData, exists := extMap["prebid"].(map[string]interface{})
	require.True(t, exists)
	targetingData, exists := prebidData["targeting"].(map[string]interface{})
	require.True(t, exists)

	// Check individual targeting keys
	assert.Equal(t, "true", targetingData["test_segment_1"])
	assert.Equal(t, "true", targetingData["test_segment_2"])
}

func TestScope3APIError(t *testing.T) {
	// Create mock server that returns an error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer mockServer.Close()

	// Create module with mock server
	config := json.RawMessage(`{
		"endpoint": "` + mockServer.URL + `",
		"auth_key": "test-auth-key",
		"timeout_ms": 1000
	}`)

	moduleInterface, err := Builder(config, getTestModuleDeps(t))
	require.NoError(t, err)
	module := moduleInterface.(*Module)

	// Test that API errors are handled gracefully
	bidRequest := &openrtb2.BidRequest{
		ID:   "test-auction",
		Site: &openrtb2.Site{Domain: "example.com"},
	}

	ctx := context.Background()
	segments, err := module.fetchScope3Segments(ctx, bidRequest)
	assert.Error(t, err)
	assert.Empty(t, segments)
	assert.Contains(t, err.Error(), "scope3 returned status 500")
}

// MASKING TESTS

func TestBuilderWithMasking(t *testing.T) {
	config := json.RawMessage(`{
		"enabled": true,
		"endpoint": "https://rtdp.scope3.com/amazonaps/rtii",
		"auth_key": "test-key",
		"timeout_ms": 1000,
		"cache_ttl_seconds": 60,
		"add_to_targeting": false,
		"masking": {
			"enabled": true,
			"geo": {
				"preserve_metro": true,
				"preserve_zip": false,
				"preserve_city": true,
				"lat_long_precision": 3
			},
			"user": {
				"preserve_eids": ["liveramp.com", "custom.com"]
			},
			"device": {
				"preserve_mobile_ids": true
			}
		}
	}`)

	deps := moduledeps.ModuleDeps{HTTPClient: http.DefaultClient}
	module, err := Builder(config, deps)

	assert.NoError(t, err)
	assert.NotNil(t, module)

	m := module.(*Module)
	assert.Equal(t, true, m.cfg.Masking.Enabled)
	assert.Equal(t, true, m.cfg.Masking.Geo.PreserveMetro)
	assert.Equal(t, false, m.cfg.Masking.Geo.PreserveZip)
	assert.Equal(t, true, m.cfg.Masking.Geo.PreserveCity)
	assert.Equal(t, 3, m.cfg.Masking.Geo.LatLongPrecision)
	assert.Equal(t, []string{"liveramp.com", "custom.com"}, m.cfg.Masking.User.PreserveEids)
	assert.Equal(t, true, m.cfg.Masking.Device.PreserveMobileIds)
}

func TestBuilderMaskingDefaults(t *testing.T) {
	config := json.RawMessage(`{
		"enabled": true,
		"auth_key": "test-key",
		"masking": {
			"enabled": true
		}
	}`)

	deps := moduledeps.ModuleDeps{HTTPClient: http.DefaultClient}
	module, err := Builder(config, deps)

	assert.NoError(t, err)
	m := module.(*Module)

	// Check defaults are applied
	assert.Equal(t, true, m.cfg.Masking.Enabled)
	assert.Equal(t, 2, m.cfg.Masking.Geo.LatLongPrecision) // Default precision
	assert.Equal(t, true, m.cfg.Masking.Geo.PreserveMetro) // Default preserve
	assert.Equal(t, true, m.cfg.Masking.Geo.PreserveZip)   // Default preserve
	assert.Equal(t, []string{"liveramp.com", "uidapi.com", "id5-sync.com"}, m.cfg.Masking.User.PreserveEids)
	assert.Equal(t, false, m.cfg.Masking.Device.PreserveMobileIds) // Default false
}

func TestMaskBidRequest_Disabled(t *testing.T) {
	module := &Module{
		cfg: Config{
			Masking: MaskingConfig{Enabled: false},
		},
	}

	lat := 37.774929
	lon := -122.419416
	original := &openrtb2.BidRequest{
		User: &openrtb2.User{
			ID: "publisher-user-123",
		},
		Device: &openrtb2.Device{
			IP:  "192.168.1.1",
			IFA: "12345-67890",
			Geo: &openrtb2.Geo{
				Lat: &lat,
				Lon: &lon,
			},
		},
	}

	result := module.maskBidRequest(original)

	// Should return exact same request when disabled
	assert.Equal(t, original, result)
}

func TestMaskUser(t *testing.T) {
	module := &Module{
		cfg: Config{
			Masking: MaskingConfig{
				Enabled: true,
				User: UserMaskingConfig{
					PreserveEids: []string{"liveramp.com", "uidapi.com"},
				},
			},
		},
	}

	original := &openrtb2.BidRequest{
		User: &openrtb2.User{
			ID:       "publisher-user-123",
			BuyerUID: "exchange-user-456",
			Yob:      1990,
			Gender:   "M",
			Keywords: "sports,automotive",
			Data: []openrtb2.Data{{
				ID:   "segment1",
				Name: "sports_fans",
			}},
			EIDs: []openrtb2.EID{
				{Source: "liveramp.com", UIDs: []openrtb2.UID{{ID: "ramp123"}}},
				{Source: "blocked.com", UIDs: []openrtb2.UID{{ID: "blocked456"}}},
				{Source: "uidapi.com", UIDs: []openrtb2.UID{{ID: "uid789"}}},
			},
		},
	}

	module.maskUser(original)

	// Check that sensitive fields are removed
	assert.Equal(t, "", original.User.ID)
	assert.Equal(t, "", original.User.BuyerUID)
	assert.Equal(t, int64(0), original.User.Yob)
	assert.Equal(t, "", original.User.Gender)
	assert.Equal(t, "", original.User.Keywords)
	assert.Nil(t, original.User.Data)

	// Check that eids are filtered correctly
	assert.Len(t, original.User.EIDs, 2)
	assert.Equal(t, "liveramp.com", original.User.EIDs[0].Source)
	assert.Equal(t, "uidapi.com", original.User.EIDs[1].Source)
}

func TestFilterEids(t *testing.T) {
	module := &Module{
		cfg: Config{
			Masking: MaskingConfig{
				User: UserMaskingConfig{
					PreserveEids: []string{"liveramp.com", "id5-sync.com"},
				},
			},
		},
	}

	eids := []openrtb2.EID{
		{Source: "liveramp.com", UIDs: []openrtb2.UID{{ID: "ramp123"}}},
		{Source: "blocked.com", UIDs: []openrtb2.UID{{ID: "blocked456"}}},
		{Source: "id5-sync.com", UIDs: []openrtb2.UID{{ID: "id5789"}}},
		{Source: "another-blocked.com", UIDs: []openrtb2.UID{{ID: "blocked999"}}},
	}

	result := module.filterEids(eids)

	assert.Len(t, result, 2)
	assert.Equal(t, "liveramp.com", result[0].Source)
	assert.Equal(t, "id5-sync.com", result[1].Source)
}

func TestFilterEids_EmptyAllowlist(t *testing.T) {
	module := &Module{
		cfg: Config{
			Masking: MaskingConfig{
				User: UserMaskingConfig{
					PreserveEids: []string{},
				},
			},
		},
	}

	eids := []openrtb2.EID{
		{Source: "liveramp.com", UIDs: []openrtb2.UID{{ID: "ramp123"}}},
		{Source: "blocked.com", UIDs: []openrtb2.UID{{ID: "blocked456"}}},
	}

	result := module.filterEids(eids)
	assert.Len(t, result, 0)
}

func TestMaskDevice(t *testing.T) {
	module := &Module{
		cfg: Config{
			Masking: MaskingConfig{
				Enabled: true,
				Device: DeviceMaskingConfig{
					PreserveMobileIds: false,
				},
			},
		},
	}

	original := &openrtb2.BidRequest{
		Device: &openrtb2.Device{
			IP:         "192.168.1.1",
			IPv6:       "2001:db8::1",
			IFA:        "12345-67890",
			DPIDMD5:    "abc123",
			DPIDSHA1:   "def456",
			DIDMD5:     "ghi789",
			DIDSHA1:    "jkl012",
			MACMD5:     "mno345",
			MACSHA1:    "pqr678",
			DeviceType: 1,
			OS:         "iOS",
			Make:       "Apple",
			Model:      "iPhone",
		},
	}

	module.maskDevice(original)

	// Check sensitive fields are removed
	assert.Equal(t, "", original.Device.IP)
	assert.Equal(t, "", original.Device.IPv6)
	assert.Equal(t, "", original.Device.IFA)
	assert.Equal(t, "", original.Device.DPIDMD5)
	assert.Equal(t, "", original.Device.DPIDSHA1)
	assert.Equal(t, "", original.Device.DIDMD5)
	assert.Equal(t, "", original.Device.DIDSHA1)
	assert.Equal(t, "", original.Device.MACMD5)
	assert.Equal(t, "", original.Device.MACSHA1)

	// Check targeting-safe fields are preserved
	assert.Equal(t, adcom1.DeviceType(1), original.Device.DeviceType)
	assert.Equal(t, "iOS", original.Device.OS)
	assert.Equal(t, "Apple", original.Device.Make)
	assert.Equal(t, "iPhone", original.Device.Model)
}

func TestMaskDevice_PreserveMobileIds(t *testing.T) {
	module := &Module{
		cfg: Config{
			Masking: MaskingConfig{
				Enabled: true,
				Device: DeviceMaskingConfig{
					PreserveMobileIds: true,
				},
			},
		},
	}

	original := &openrtb2.BidRequest{
		Device: &openrtb2.Device{
			IP:       "192.168.1.1",
			IFA:      "12345-67890",
			DPIDMD5:  "abc123",
			DPIDSHA1: "def456",
		},
	}

	module.maskDevice(original)

	// IP should still be removed
	assert.Equal(t, "", original.Device.IP)

	// But mobile IDs should be preserved
	assert.Equal(t, "12345-67890", original.Device.IFA)
	assert.Equal(t, "abc123", original.Device.DPIDMD5)
	assert.Equal(t, "def456", original.Device.DPIDSHA1)
}

func TestMaskGeoObject(t *testing.T) {
	lat := 37.774929
	lon := -122.419416
	accuracy := int64(10)

	tests := []struct {
		name             string
		config           GeoMaskingConfig
		expectedLat      *float64
		expectedLon      *float64
		expectedMetro    string
		expectedZip      string
		expectedCity     string
		expectedAccuracy int64
	}{
		{
			name: "precision_0_removes_coordinates",
			config: GeoMaskingConfig{
				LatLongPrecision: 0,
				PreserveMetro:    true,
				PreserveZip:      true,
				PreserveCity:     true,
			},
			expectedLat:      nil,
			expectedLon:      nil,
			expectedMetro:    "807",           // preserved
			expectedZip:      "94102",         // preserved
			expectedCity:     "San Francisco", // preserved
			expectedAccuracy: 0,               // always removed
		},
		{
			name: "precision_2_truncates_coordinates",
			config: GeoMaskingConfig{
				LatLongPrecision: 2,
				PreserveMetro:    false,
				PreserveZip:      false,
				PreserveCity:     false,
			},
			expectedLat:      &[]float64{37.77}[0],   // truncated to 2 decimals
			expectedLon:      &[]float64{-122.41}[0], // truncated to 2 decimals
			expectedMetro:    "",                     // not preserved
			expectedZip:      "",                     // not preserved
			expectedCity:     "",                     // not preserved
			expectedAccuracy: 0,                      // always removed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module := &Module{
				cfg: Config{
					Masking: MaskingConfig{
						Enabled: true,
						Geo:     tt.config,
					},
				},
			}

			geo := &openrtb2.Geo{
				Lat:      &lat,
				Lon:      &lon,
				Country:  "USA", // always preserved
				Region:   "CA",  // always preserved
				Metro:    "807",
				ZIP:      "94102",
				City:     "San Francisco",
				Accuracy: accuracy,
			}

			module.maskGeoObject(geo)

			// Check coordinates
			if tt.expectedLat == nil {
				assert.Nil(t, geo.Lat)
			} else {
				require.NotNil(t, geo.Lat)
				assert.InDelta(t, *tt.expectedLat, *geo.Lat, 0.001)
			}

			if tt.expectedLon == nil {
				assert.Nil(t, geo.Lon)
			} else {
				require.NotNil(t, geo.Lon)
				assert.InDelta(t, *tt.expectedLon, *geo.Lon, 0.001)
			}

			// Check other fields
			assert.Equal(t, "USA", geo.Country) // always preserved
			assert.Equal(t, "CA", geo.Region)   // always preserved
			assert.Equal(t, tt.expectedMetro, geo.Metro)
			assert.Equal(t, tt.expectedZip, geo.ZIP)
			assert.Equal(t, tt.expectedCity, geo.City)
			assert.Equal(t, int64(0), geo.Accuracy) // always removed
		})
	}
}

func TestTruncateCoordinate(t *testing.T) {
	module := &Module{}

	tests := []struct {
		coord     float64
		precision int
		expected  float64
	}{
		{37.774929, 0, 0},
		{37.774929, 1, 37.7},
		{37.774929, 2, 37.77},
		{37.774929, 3, 37.774},
		{37.774929, 4, 37.7749},
		{-122.419416, 2, -122.41},
		{-122.419416, 3, -122.419},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("coord_%.6f_precision_%d", tt.coord, tt.precision), func(t *testing.T) {
			result := module.truncateCoordinate(tt.coord, tt.precision)
			assert.InDelta(t, tt.expected, result, 0.0001)
		})
	}
}

func TestMaskBidRequest_FullIntegration(t *testing.T) {
	module := &Module{
		cfg: Config{
			Masking: MaskingConfig{
				Enabled: true,
				Geo: GeoMaskingConfig{
					PreserveMetro:    true,
					PreserveZip:      false,
					PreserveCity:     false,
					LatLongPrecision: 2,
				},
				User: UserMaskingConfig{
					PreserveEids: []string{"liveramp.com"},
				},
				Device: DeviceMaskingConfig{
					PreserveMobileIds: false,
				},
			},
		},
	}

	lat := 37.774929
	lon := -122.419416
	original := &openrtb2.BidRequest{
		ID: "test-request",
		User: &openrtb2.User{
			ID:       "publisher-user-123",
			BuyerUID: "exchange-user-456",
			Gender:   "M",
			EIDs: []openrtb2.EID{
				{Source: "liveramp.com", UIDs: []openrtb2.UID{{ID: "ramp123"}}},
				{Source: "blocked.com", UIDs: []openrtb2.UID{{ID: "blocked456"}}},
			},
		},
		Device: &openrtb2.Device{
			IP:         "192.168.1.1",
			IFA:        "12345-67890",
			DeviceType: 1,
			OS:         "iOS",
			Geo: &openrtb2.Geo{
				Lat:     &lat,
				Lon:     &lon,
				Country: "USA",
				Region:  "CA",
				Metro:   "807",
				ZIP:     "94102",
				City:    "San Francisco",
			},
		},
		Site: &openrtb2.Site{
			Domain: "example.com",
			Page:   "https://example.com/test",
		},
	}

	result := module.maskBidRequest(original)

	// Should be different object (deep copy)
	assert.NotEqual(t, original, result)

	// Check user masking
	assert.Equal(t, "", result.User.ID)
	assert.Equal(t, "", result.User.BuyerUID)
	assert.Equal(t, "", result.User.Gender)
	assert.Len(t, result.User.EIDs, 1)
	assert.Equal(t, "liveramp.com", result.User.EIDs[0].Source)

	// Check device masking
	assert.Equal(t, "", result.Device.IP)
	assert.Equal(t, "", result.Device.IFA)
	assert.Equal(t, adcom1.DeviceType(1), result.Device.DeviceType) // preserved
	assert.Equal(t, "iOS", result.Device.OS)                        // preserved

	// Check geo masking
	assert.Equal(t, "USA", result.Device.Geo.Country)         // preserved
	assert.Equal(t, "CA", result.Device.Geo.Region)           // preserved
	assert.Equal(t, "807", result.Device.Geo.Metro)           // preserved
	assert.Equal(t, "", result.Device.Geo.ZIP)                // not preserved
	assert.Equal(t, "", result.Device.Geo.City)               // not preserved
	assert.InDelta(t, 37.77, *result.Device.Geo.Lat, 0.001)   // truncated
	assert.InDelta(t, -122.41, *result.Device.Geo.Lon, 0.001) // truncated

	// Check site is preserved completely
	assert.Equal(t, "example.com", result.Site.Domain)
	assert.Equal(t, "https://example.com/test", result.Site.Page)

	// Original should be unchanged
	assert.Equal(t, "publisher-user-123", original.User.ID)
	assert.Equal(t, "192.168.1.1", original.Device.IP)
	assert.InDelta(t, 37.774929, *original.Device.Geo.Lat, 0.000001)
}

func TestGetMaskingSummary(t *testing.T) {
	module := &Module{
		cfg: Config{
			Masking: MaskingConfig{
				Enabled: true,
				Geo: GeoMaskingConfig{
					PreserveMetro:    true,
					PreserveZip:      false,
					LatLongPrecision: 3,
				},
				User: UserMaskingConfig{
					PreserveEids: []string{"liveramp.com", "uidapi.com"},
				},
				Device: DeviceMaskingConfig{
					PreserveMobileIds: true,
				},
			},
		},
	}

	summary := module.getMaskingSummary()

	assert.Equal(t, true, summary["enabled"])

	geoConfig, ok := summary["geo"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, geoConfig["preserve_metro"])
	assert.Equal(t, false, geoConfig["preserve_zip"])
	assert.Equal(t, 3, geoConfig["lat_long_precision"])

	userConfig, ok := summary["user"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, []string{"liveramp.com", "uidapi.com"}, userConfig["preserve_eids"])

	deviceConfig, ok := summary["device"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, deviceConfig["preserve_mobile_ids"])
}

func TestGetMaskingSummary_Disabled(t *testing.T) {
	module := &Module{
		cfg: Config{
			Masking: MaskingConfig{Enabled: false},
		},
	}

	summary := module.getMaskingSummary()
	assert.Equal(t, map[string]interface{}{"enabled": false}, summary)
}

func TestMaskGeo_UserGeoOnly(t *testing.T) {
	module := &Module{
		cfg: Config{
			Masking: MaskingConfig{
				Enabled: true,
				Geo: GeoMaskingConfig{
					PreserveMetro:    false,
					PreserveZip:      false,
					LatLongPrecision: 2,
				},
			},
		},
	}

	lat := 40.7128
	lon := -74.0060
	original := &openrtb2.BidRequest{
		User: &openrtb2.User{
			Geo: &openrtb2.Geo{
				Lat:     &lat,
				Lon:     &lon,
				Country: "USA",
				Region:  "NY",
				Metro:   "501",
				ZIP:     "10001",
			},
		},
		// No device geo
		Device: &openrtb2.Device{
			OS: "iOS",
		},
	}

	module.maskGeo(original)

	// Check user geo was masked
	assert.Equal(t, "USA", original.User.Geo.Country)        // preserved
	assert.Equal(t, "NY", original.User.Geo.Region)          // preserved
	assert.Equal(t, "", original.User.Geo.Metro)             // removed
	assert.Equal(t, "", original.User.Geo.ZIP)               // removed
	assert.InDelta(t, 40.71, *original.User.Geo.Lat, 0.001)  // truncated
	assert.InDelta(t, -74.00, *original.User.Geo.Lon, 0.001) // truncated
}

func TestTruncateCoordinate_EdgeCases(t *testing.T) {
	module := &Module{}

	// Test precision out of range
	assert.Equal(t, float64(0), module.truncateCoordinate(37.774929, -1))
	assert.Equal(t, float64(0), module.truncateCoordinate(37.774929, 5))
	assert.Equal(t, float64(0), module.truncateCoordinate(37.774929, 0))

	// Test zero coordinate
	assert.Equal(t, float64(0), module.truncateCoordinate(0.0, 2))
}

func TestMaskDevice_NoDevice(t *testing.T) {
	module := &Module{
		cfg: Config{
			Masking: MaskingConfig{Enabled: true},
		},
	}

	original := &openrtb2.BidRequest{
		ID:   "test",
		User: &openrtb2.User{ID: "user-123"},
		// No device
	}

	module.maskDevice(original)

	// Should not crash when device is nil
	assert.Nil(t, original.Device)
}

func TestMaskUser_NoUser(t *testing.T) {
	module := &Module{
		cfg: Config{
			Masking: MaskingConfig{Enabled: true},
		},
	}

	original := &openrtb2.BidRequest{
		ID: "test",
		// No user
		Device: &openrtb2.Device{IP: "192.168.1.1"},
	}

	module.maskUser(original)

	// Should not crash when user is nil
	assert.Nil(t, original.User)
}

func TestBuilderConfigValidation_GeoPrecisionTooHigh(t *testing.T) {
	config := json.RawMessage(`{
		"enabled": true,
		"auth_key": "test-key",
		"masking": {
			"enabled": true,
			"geo": {
				"lat_long_precision": 5
			}
		}
	}`)

	deps := moduledeps.ModuleDeps{HTTPClient: http.DefaultClient}
	module, err := Builder(config, deps)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot exceed 4 decimal places")
	assert.Nil(t, module)
}

func TestBuilderConfigValidation_GeoPrecisionNegative(t *testing.T) {
	config := json.RawMessage(`{
		"enabled": true,
		"auth_key": "test-key",
		"masking": {
			"enabled": true,
			"geo": {
				"lat_long_precision": -1
			}
		}
	}`)

	deps := moduledeps.ModuleDeps{HTTPClient: http.DefaultClient}
	module, err := Builder(config, deps)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be negative")
	assert.Nil(t, module)
}

func TestBuilderConfigValidation_GeoPrecisionValid(t *testing.T) {
	config := json.RawMessage(`{
		"enabled": true,
		"auth_key": "test-key",
		"masking": {
			"enabled": true,
			"geo": {
				"lat_long_precision": 4
			}
		}
	}`)

	deps := moduledeps.ModuleDeps{HTTPClient: http.DefaultClient}
	module, err := Builder(config, deps)

	assert.NoError(t, err)
	assert.NotNil(t, module)

	m := module.(*Module)
	assert.Equal(t, 4, m.cfg.Masking.Geo.LatLongPrecision)
}

func TestCreateCacheKey_HashedUserID(t *testing.T) {
	module := &Module{
		sha256Pool: &sync.Pool{
			New: func() any {
				return sha256.New()
			},
		},
	}

	// Create request with user ID (no privacy-safe identifiers)
	bidRequest := &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Domain: "example.com",
			Page:   "https://example.com/test",
		},
		User: &openrtb2.User{
			ID: "sensitive-user-id-123", // This should be hashed in cache key
		},
	}

	key1 := module.createCacheKey(bidRequest)

	// Create another request with different user ID
	bidRequest2 := &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Domain: "example.com",
			Page:   "https://example.com/test",
		},
		User: &openrtb2.User{
			ID: "different-user-id-456", // Different hash should produce different key
		},
	}

	key2 := module.createCacheKey(bidRequest2)

	// Keys should be different since user IDs are different (per-user caching)
	assert.NotEqual(t, key1, key2, "Cache keys should be different for different user IDs to enable per-user caching")

	// Create request with same user ID to verify consistency
	bidRequest3 := &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Domain: "example.com",
			Page:   "https://example.com/test",
		},
		User: &openrtb2.User{
			ID: "sensitive-user-id-123", // Same as first request
		},
	}

	key3 := module.createCacheKey(bidRequest3)

	// Should match first key for same user
	assert.Equal(t, key1, key3, "Cache keys should be consistent for same user ID")

	// Verify key is SHA-256 length (64 hex characters)
	assert.Len(t, key1, 64, "Cache key should be SHA-256 hash (64 characters)")
}

func TestCreateCacheKey_PrivacySafeIdentifiersPriority(t *testing.T) {
	module := &Module{
		sha256Pool: &sync.Pool{
			New: func() any {
				return sha256.New()
			},
		},
	}

	// Create request with both user.id and privacy-safe identifiers
	userExtBytes, _ := jsonutil.Marshal(userExt{
		RampID: "ramp_abc123",
	})

	bidRequest := &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Domain: "example.com",
			Page:   "https://example.com/test",
		},
		User: &openrtb2.User{
			ID:  "user-id-should-not-be-used",
			Ext: userExtBytes,
		},
	}

	key1 := module.createCacheKey(bidRequest)

	// Create another request with same privacy-safe ID but different user.id
	bidRequest2 := &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Domain: "example.com",
			Page:   "https://example.com/test",
		},
		User: &openrtb2.User{
			ID:  "completely-different-user-id",
			Ext: userExtBytes, // Same RampID
		},
	}

	key2 := module.createCacheKey(bidRequest2)

	// Keys should be the same since privacy-safe identifiers take priority
	assert.Equal(t, key1, key2, "Cache keys should be same when privacy-safe identifiers match, regardless of user.id")
}

func TestFetchScope3Segments_MaskingFailure(t *testing.T) {
	// Create a test server that simulates API response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{"data": []}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
	defer mockServer.Close()

	module := &Module{
		cfg: Config{
			Endpoint: mockServer.URL,
			AuthKey:  "test-key",
			Timeout:  1000,
			Masking:  MaskingConfig{Enabled: true},
		},
		httpClient: &http.Client{
			Timeout: 1 * time.Second,
		},
		cache: freecache.NewCache(10),
		sha256Pool: &sync.Pool{
			New: func() any {
				return sha256.New()
			},
		},
	}

	// Create a request that should work normally
	bidRequest := &openrtb2.BidRequest{
		ID: "test-request",
		User: &openrtb2.User{
			ID: "user-123",
		},
	}

	ctx := context.Background()

	// This should succeed now that we have proper HTTP client setup
	// The masking should work correctly and not cause errors
	segments, err := module.fetchScope3Segments(ctx, bidRequest)

	// Should succeed with proper setup
	assert.NoError(t, err)
	assert.NotNil(t, segments)
}
