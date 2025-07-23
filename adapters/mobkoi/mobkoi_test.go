package mobkoi

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// The endpoint of that the bid requests are sent to.
const ADAPTER_BIDDER_ENDPOINT = "https://adapter.config.bidder.endpoint.com/bid"
// The public-facing URL of the Prebid Server instance.
const BIDDER_EXTERNAL_URL = "https://prebid.server.test.com"

func getValidAdapterConfig() config.Adapter {
	return config.Adapter{
		Endpoint: ADAPTER_BIDDER_ENDPOINT,
	}
}

func getValidServerConfig() config.Server {
	return config.Server{
	ExternalUrl: BIDDER_EXTERNAL_URL,
	GvlID:       1,
	DataCenter:  "2",
	}
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderMobkoi,
		getValidAdapterConfig(),
		getValidServerConfig(),
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "mobkoitest", bidder)
}

func TestIntegrationTypeIsSet(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderMobkoi,
		getValidAdapterConfig(),
		getValidServerConfig(),
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create test request with empty ext
	testRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID:    "test-imp-id",
				TagID: "test-tag-id",
				Ext:   json.RawMessage(`{"bidder": {"placementId": "test-placement", "integrationEndpoint": "http://test.mobkoi.com/bid"}}`),
			},
		},
	}

	requestData, errs := bidder.MakeRequests(testRequest, &adapters.ExtraRequestInfo{})

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got %v", errs)
	}

	if len(requestData) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requestData))
	}

	// Parse the modified request from the body
	var modifiedRequest openrtb2.BidRequest
	if err := json.Unmarshal(requestData[0].Body, &modifiedRequest); err != nil {
		t.Fatalf("Failed to unmarshal request body: %v", err)
	}

	// Verify request-level ext.mobkoi.integration_type is set
	if modifiedRequest.Ext == nil {
		t.Fatal("Expected request.Ext to be set")
	}

	// Parse request extension as map to check mobkoi extension
	extMap := make(map[string]json.RawMessage)
	if err := json.Unmarshal(modifiedRequest.Ext, &extMap); err != nil {
		t.Fatalf("Failed to unmarshal request.Ext: %v", err)
	}

	mobkoiExtBytes, exists := extMap["mobkoi"]
	if !exists {
		t.Fatal("Expected request.Ext.mobkoi to be set")
	}

	var mobkoiExt map[string]interface{}
	if err := json.Unmarshal(mobkoiExtBytes, &mobkoiExt); err != nil {
		t.Fatalf("Failed to unmarshal mobkoi extension: %v", err)
	}

	if integrationType, ok := mobkoiExt["integration_type"]; !ok || integrationType != "pbs" {
		t.Errorf("Expected integration_type to be 'pbs', got '%v'", integrationType)
	}
}

func TestIntegrationTypeWithExistingRequestExt(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderMobkoi,
		getValidAdapterConfig(),
		getValidServerConfig(),
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create test request with existing request-level extension
	existingExt := openrtb_ext.ExtRequest{}
	extBytes, _ := json.Marshal(existingExt)

	testRequest := &openrtb2.BidRequest{
		ID:  "test-request-id",
		Ext: extBytes,
		Imp: []openrtb2.Imp{
			{
				ID:    "test-imp-id",
				TagID: "test-tag-id",
				Ext:   json.RawMessage(`{"bidder": {"placementId": "test-placement"}}`),
			},
		},
	}

	requestData, errs := bidder.MakeRequests(testRequest, &adapters.ExtraRequestInfo{})

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got %v", errs)
	}

	// Parse the modified request from the body
	var modifiedRequest openrtb2.BidRequest
	if err := json.Unmarshal(requestData[0].Body, &modifiedRequest); err != nil {
		t.Fatalf("Failed to unmarshal request body: %v", err)
	}

	// Verify both existing and new extensions are preserved
	extMap := make(map[string]json.RawMessage)
	if err := json.Unmarshal(modifiedRequest.Ext, &extMap); err != nil {
		t.Fatalf("Failed to unmarshal request.Ext: %v", err)
	}

	// Check new mobkoi extension is added
	mobkoiExtBytes, exists := extMap["mobkoi"]
	if !exists {
		t.Fatal("Expected request.Ext.mobkoi to be set")
	}

	var mobkoiExt map[string]interface{}
	if err := json.Unmarshal(mobkoiExtBytes, &mobkoiExt); err != nil {
		t.Fatalf("Failed to unmarshal mobkoi extension: %v", err)
	}

	if integrationType, ok := mobkoiExt["integration_type"]; !ok || integrationType != "pbs" {
		t.Errorf("Expected integration_type to be 'pbs', got '%v'", integrationType)
	}
}

func TestFallbackToAdapterBidderEndpoint(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderMobkoi,
		getValidAdapterConfig(),
		getValidServerConfig(),
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create test request with NO integrationEndpoint in imp.ext.bidder - should fall back to adapter bidder endpoint
	testRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID:    "test-imp-id",
				TagID: "test-tag-id",
				Ext:   json.RawMessage(`{"bidder": {"placementId": "test-placement"}}`), // No integrationEndpoint specified
			},
		},
	}

	requestData, errs := bidder.MakeRequests(testRequest, &adapters.ExtraRequestInfo{})

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got %v", errs)
	}

	if len(requestData) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requestData))
	}

	// Should use adapter bidder endpoint since no integration endpoint provided
	if requestData[0].Uri != ADAPTER_BIDDER_ENDPOINT {
		t.Errorf("Expected endpoint to be '%s' (adapter bidder endpoint), got '%s'", ADAPTER_BIDDER_ENDPOINT, requestData[0].Uri)
	}
}

func TestEndpointFromBidderExt(t *testing.T) {
	shouldNotBeUsedEndpoint := "https://should.not.be.used.adapter.bidder.endpoint.com" // Should not be used since integration endpoint is provided
	
	bidder, buildErr := Builder(
		openrtb_ext.BidderMobkoi,
		config.Adapter{
			Endpoint: shouldNotBeUsedEndpoint,
		},
		getValidServerConfig(),
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create test request with integrationEndpoint in imp.ext.bidder - should take precedence over adapter bidder endpoint
	testRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID:    "test-imp-id",
				TagID: "test-tag-id",
				Ext:   json.RawMessage(fmt.Sprintf(`{"bidder": {"placementId": "test-placement", "integrationEndpoint": "%s"}}`, ADAPTER_BIDDER_ENDPOINT)),
			},
		},
	}

	requestData, errs := bidder.MakeRequests(testRequest, &adapters.ExtraRequestInfo{})

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got %v", errs)
	}

	if len(requestData) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requestData))
	}

	// Should use integration endpoint from bidder extension, not adapter bidder endpoint
	if requestData[0].Uri != ADAPTER_BIDDER_ENDPOINT {
		t.Errorf("Expected endpoint to be '%s' (integration endpoint from bidder ext), got '%s'", ADAPTER_BIDDER_ENDPOINT, requestData[0].Uri)
	}
}

func TestErrorWhenNoValidEndpoints(t *testing.T) {
	invalidAdapterBidderEndpoint := "invalid-adapter-bidder-url" // Invalid adapter bidder endpoint
	invalidIntegrationEndpointFromBidderExt := "not-a-valid-integration-endpoint-url" // Invalid integration endpoint from bidder extension
	
	bidder, buildErr := Builder(
		openrtb_ext.BidderMobkoi,
		config.Adapter{
			Endpoint: invalidAdapterBidderEndpoint, // Invalid adapter bidder endpoint from config
		},
		getValidServerConfig(),
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create test request with invalid integrationEndpoint - both endpoints are invalid, should return error
	testRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID:    "test-imp-id",
				TagID: "test-tag-id",
				Ext:   json.RawMessage(fmt.Sprintf(`{"bidder": {"placementId": "test-placement", "integrationEndpoint": "%s"}}`, invalidIntegrationEndpointFromBidderExt)),
			},
		},
	}

	requestData, errs := bidder.MakeRequests(testRequest, &adapters.ExtraRequestInfo{})

	if len(errs) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errs))
	}

	if len(requestData) != 0 {
		t.Fatalf("Expected 0 request data, got %d", len(requestData))
	}

	expectedError := fmt.Sprintf("no valid endpoint configured: both integration endpoint (%s) and bidder endpoint (%s) are invalid", invalidIntegrationEndpointFromBidderExt, invalidAdapterBidderEndpoint)
	if errs[0].Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, errs[0].Error())
	}
}

func TestErrorWhenEmptyEndpoints(t *testing.T) {
	emptyAdapterBidderEndpoint := "" // Empty adapter bidder endpoint from config
	emptyIntegrationEndpoint := "" // Empty integration endpoint (none provided)
	
	bidder, buildErr := Builder(
		openrtb_ext.BidderMobkoi,
		config.Adapter{
			Endpoint: emptyAdapterBidderEndpoint,
		},
		getValidServerConfig(),
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create test request with no integrationEndpoint - both endpoints are empty, should return error
	testRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID:    "test-imp-id",
				TagID: "test-tag-id",
				Ext:   json.RawMessage(`{"bidder": {"placementId": "test-placement"}}`), // No integrationEndpoint specified
			},
		},
	}

	requestData, errs := bidder.MakeRequests(testRequest, &adapters.ExtraRequestInfo{})

	if len(errs) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errs))
	}

	if len(requestData) != 0 {
		t.Fatalf("Expected 0 request data, got %d", len(requestData))
	}

	expectedError := fmt.Sprintf("no valid endpoint configured: both integration endpoint (%s) and bidder endpoint (%s) are invalid", emptyIntegrationEndpoint, emptyAdapterBidderEndpoint)
	if errs[0].Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, errs[0].Error())
	}
}
