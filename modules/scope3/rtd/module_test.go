package scope3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	config := json.RawMessage(`{
		"enabled": true,
		"endpoint": "https://rtdp.scope3.com/amazonaps/rtii",
		"auth_key": "test-key",
		"timeout_ms": 1000,
		"cache_ttl_seconds": 60,
		"add_to_targeting": false
	}`)

	deps := moduledeps.ModuleDeps{}
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
	deps := moduledeps.ModuleDeps{}

	module, err := Builder(config, deps)

	assert.Error(t, err)
	assert.Nil(t, module)
}

func TestHandleEntrypointHook(t *testing.T) {
	module := &Module{}
	ctx := context.Background()
	miCtx := hookstage.ModuleInvocationContext{}
	payload := hookstage.EntrypointPayload{}

	result, err := module.HandleEntrypointHook(ctx, miCtx, payload)

	assert.NoError(t, err)
	assert.NotNil(t, result.ModuleContext["segments"])
}

func TestHandleAuctionResponseHook_NoSegments(t *testing.T) {
	module := &Module{}
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

	deps := moduledeps.ModuleDeps{}
	module, err := Builder(config, deps)

	assert.NoError(t, err)
	m := module.(*Module)
	assert.Equal(t, "https://rtdp.scope3.com/prebid/rtii", m.cfg.Endpoint)
	assert.Equal(t, 1000, m.cfg.Timeout)
	assert.Equal(t, 60, m.cfg.CacheTTL)
	assert.Equal(t, false, m.cfg.AddToTargeting)
}

func TestHTTPTransportOptimization(t *testing.T) {
	config := json.RawMessage(`{
		"enabled": true,
		"auth_key": "test-key",
		"timeout_ms": 2000
	}`)

	deps := moduledeps.ModuleDeps{}
	module, err := Builder(config, deps)

	assert.NoError(t, err)
	m := module.(*Module)

	// Verify HTTP client configuration
	assert.NotNil(t, m.httpClient)
	assert.Equal(t, 2000*time.Millisecond, m.httpClient.Timeout)

	// Verify transport is configured for high-frequency requests
	transport, ok := m.httpClient.Transport.(*http.Transport)
	require.True(t, ok, "Expected custom HTTP transport")

	assert.Equal(t, 100, transport.MaxIdleConns)
	assert.Equal(t, 10, transport.MaxIdleConnsPerHost)
	assert.Equal(t, 90*time.Second, transport.IdleConnTimeout)
	assert.Equal(t, false, transport.DisableCompression)
	assert.Equal(t, true, transport.ForceAttemptHTTP2)
}

func TestCacheOperations(t *testing.T) {
	cache := &segmentCache{data: make(map[string]cacheEntry)}

	// Test cache miss
	segments, found := cache.get("test-key", time.Minute)
	assert.False(t, found)
	assert.Nil(t, segments)

	// Test cache set and hit
	testSegments := []string{"segment1", "segment2"}
	cache.set("test-key", testSegments)

	segments, found = cache.get("test-key", time.Minute)
	assert.True(t, found)
	assert.Equal(t, testSegments, segments)

	// Test cache expiry
	segments, found = cache.get("test-key", time.Nanosecond)
	assert.False(t, found)
	assert.Nil(t, segments)
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
		w.Write([]byte(response))
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

	deps := moduledeps.ModuleDeps{}
	moduleInterface, err := Builder(config, deps)
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
		w.Write([]byte(response))
	}))
	defer mockServer.Close()

	// Create module with targeting enabled
	config := json.RawMessage(`{
		"endpoint": "` + mockServer.URL + `",
		"auth_key": "test-auth-key",
		"timeout_ms": 1000,
		"add_to_targeting": true
	}`)

	deps := moduledeps.ModuleDeps{}
	moduleInterface, err := Builder(config, deps)
	require.NoError(t, err)
	module := moduleInterface.(*Module)

	// Test full hook workflow
	ctx := context.Background()

	// Test entrypoint hook
	entrypointResult, err := module.HandleEntrypointHook(ctx, hookstage.ModuleInvocationContext{}, hookstage.EntrypointPayload{})
	require.NoError(t, err)
	assert.NotNil(t, entrypointResult.ModuleContext["segments"])

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
		w.Write([]byte(response))
	}))
	defer mockServer.Close()

	// Create module with targeting enabled
	config := json.RawMessage(`{
		"endpoint": "` + mockServer.URL + `",
		"auth_key": "test-auth-key",
		"timeout_ms": 1000,
		"add_to_targeting": true
	}`)

	deps := moduledeps.ModuleDeps{}
	moduleInterface, err := Builder(config, deps)
	require.NoError(t, err)
	module := moduleInterface.(*Module)

	// Test full hook workflow
	ctx := context.Background()

	// Test entrypoint hook
	entrypointResult, err := module.HandleEntrypointHook(ctx, hookstage.ModuleInvocationContext{}, hookstage.EntrypointPayload{})
	require.NoError(t, err)
	assert.NotNil(t, entrypointResult.ModuleContext["segments"])

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
		w.Write([]byte(response))
	}))
	defer mockServer.Close()

	// Create module with targeting enabled
	config := json.RawMessage(`{
		"endpoint": "` + mockServer.URL + `",
		"auth_key": "test-auth-key",
		"timeout_ms": 1000,
		"add_to_targeting": true
	}`)

	deps := moduledeps.ModuleDeps{}
	moduleInterface, err := Builder(config, deps)
	require.NoError(t, err)
	module := moduleInterface.(*Module)

	// Test full hook workflow
	ctx := context.Background()

	// Test entrypoint hook
	entrypointResult, err := module.HandleEntrypointHook(ctx, hookstage.ModuleInvocationContext{}, hookstage.EntrypointPayload{})
	require.NoError(t, err)
	assert.NotNil(t, entrypointResult.ModuleContext["segments"])

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
		w.Write([]byte("Internal Server Error"))
	}))
	defer mockServer.Close()

	// Create module with mock server
	config := json.RawMessage(`{
		"endpoint": "` + mockServer.URL + `",
		"auth_key": "test-auth-key",
		"timeout_ms": 1000
	}`)

	deps := moduledeps.ModuleDeps{}
	moduleInterface, err := Builder(config, deps)
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
