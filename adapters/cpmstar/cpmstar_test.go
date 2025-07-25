package cpmstar

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderCpmstar, config.Adapter{
		Endpoint: "//host"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "cpmstartest", bidder)
}

// TestExtensionFieldPreservation tests that the adapter preserves all fields in imp.ext
// including gpid and other extension fields, and does not inadvertently remove them
func TestExtensionFieldPreservation(t *testing.T) {
	adapter := &Adapter{endpoint: "http://test.endpoint"}

	// Create a bid request with imp.ext containing multiple fields including gpid
	bidRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID: "test-imp-id",
				Banner: &openrtb2.Banner{
					W: openrtb2.Int64Ptr(300),
					H: openrtb2.Int64Ptr(250),
				},
				Ext: json.RawMessage(`{
					"gpid": "/homepage/banner1",
					"prebid": {
						"storedrequest": {
							"id": "stored-imp-id"
						}
					},
					"custom_field": "custom_value",
					"bidder": {
						"placementId": 154,
						"subpoolId": 123
					}
				}`),
			},
		},
	}

	// Make the request
	requests, errs := adapter.MakeRequests(bidRequest, &adapters.ExtraRequestInfo{})

	// Should not have errors
	assert.Empty(t, errs, "Should not have errors")
	assert.Len(t, requests, 1, "Should have exactly one request")

	if len(requests) == 0 {
		t.Fatal("No requests generated")
	}

	// Parse the generated request body
	var processedRequest openrtb2.BidRequest
	err := json.Unmarshal(requests[0].Body, &processedRequest)
	require.NoError(t, err, "Should be able to unmarshal the request body")

	// Verify we have the expected imp
	assert.Len(t, processedRequest.Imp, 1, "Should have exactly one impression")

	// Parse the extension from the processed request
	var processedExt map[string]interface{}
	err = json.Unmarshal(processedRequest.Imp[0].Ext, &processedExt)
	require.NoError(t, err, "Should be able to unmarshal the processed extension")

	// Verify that all original fields are preserved
	assert.Equal(t, "/homepage/banner1", processedExt["gpid"], "gpid field should be preserved")
	assert.NotNil(t, processedExt["prebid"], "prebid field should be preserved")
	assert.Equal(t, "custom_value", processedExt["custom_field"], "custom_field should be preserved")

	// Verify the bidder-specific config is flattened to root level (not nested under 'bidder')
	assert.Nil(t, processedExt["bidder"], "bidder wrapper should be removed")
	assert.Equal(t, float64(154), processedExt["placementId"], "placementId should be at root level")
	assert.Equal(t, float64(123), processedExt["subpoolId"], "subpoolId should be at root level")
}

// TestExtensionFieldPreservationMultipleImps tests field preservation with multiple impressions
func TestExtensionFieldPreservationMultipleImps(t *testing.T) {
	adapter := &Adapter{endpoint: "http://test.endpoint"}

	bidRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID:     "test-imp-id-1",
				Banner: &openrtb2.Banner{W: openrtb2.Int64Ptr(300), H: openrtb2.Int64Ptr(250)},
				Ext: json.RawMessage(`{
					"gpid": "/homepage/banner1",
					"bidder": {"placementId": 154}
				}`),
			},
			{
				ID:    "test-imp-id-2",
				Video: &openrtb2.Video{W: openrtb2.Int64Ptr(640), H: openrtb2.Int64Ptr(480)},
				Ext: json.RawMessage(`{
					"gpid": "/homepage/video1",
					"schain": {"complete": 1},
					"bidder": {"placementId": 155}
				}`),
			},
		},
	}

	requests, errs := adapter.MakeRequests(bidRequest, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs, "Should not have errors")
	assert.Len(t, requests, 1, "Should have exactly one request")

	var processedRequest openrtb2.BidRequest
	err := json.Unmarshal(requests[0].Body, &processedRequest)
	require.NoError(t, err)

	assert.Len(t, processedRequest.Imp, 2, "Should have two impressions")

	// Check first impression
	var ext1 map[string]interface{}
	err = json.Unmarshal(processedRequest.Imp[0].Ext, &ext1)
	require.NoError(t, err)
	assert.Equal(t, "/homepage/banner1", ext1["gpid"], "First imp gpid should be preserved")

	// Check second impression
	var ext2 map[string]interface{}
	err = json.Unmarshal(processedRequest.Imp[1].Ext, &ext2)
	require.NoError(t, err)
	assert.Equal(t, "/homepage/video1", ext2["gpid"], "Second imp gpid should be preserved")
	assert.NotNil(t, ext2["schain"], "schain field should be preserved")
}