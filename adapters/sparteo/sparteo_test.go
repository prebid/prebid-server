package sparteo

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuilder verifies that the Builder function correctly creates a bidder instance.
// It checks for errors, ensures the returned bidder is not nil, and confirms that the endpoint
// in the adapter is set according to the configuration.
func TestBuilder(t *testing.T) {
	cfg := config.Adapter{Endpoint: "https://bid-test.sparteo.com/s2s-auction"}
	bidder, err := Builder(openrtb_ext.BidderSparteo, cfg, config.Server{GvlID: 1028})

	require.NoError(t, err, "Builder returned an error")
	assert.NotNil(t, bidder, "Bidder is nil")

	sparteoAdapter, ok := bidder.(*adapter)
	require.True(t, ok, "Expected *adapter, got %T", bidder)

	assert.Equal(t, "https://bid-test.sparteo.com/s2s-auction", sparteoAdapter.endpoint, "Endpoint is not correctly set")
}

// TestJsonSamples runs JSON sample tests using the shared adapterstest framework.
func TestJsonSamples(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{
		Endpoint: "https://bid-test.sparteo.com/s2s-auction",
	}, config.Server{GvlID: 1028})
	require.NoError(t, err, "Builder returned an error")

	adapterstest.RunJSONBidderTest(t, "sparteotest", bidder)
}

// TestGetMediaType_InvalidJSON verifies that getMediaType returns an error and an empty result
// when the extension JSON is invalid.
func TestGetMediaType_InvalidJSON(t *testing.T) {
	adapter := &adapter{}
	bid := &openrtb2.Bid{
		Ext: json.RawMessage(`invalid-json`),
	}
	result, err := adapter.getMediaType(bid)
	assert.Error(t, err, "Expected error for invalid JSON")
	assert.Equal(t, openrtb_ext.BidType(""), result, "Expected empty result for invalid JSON")
}

// TestGetMediaType_EmptyType verifies that getMediaType returns an error and an empty result
// when the extension JSON is valid but the "type" field is empty.
func TestGetMediaType_EmptyType(t *testing.T) {
	adapter := &adapter{}
	bid := &openrtb2.Bid{
		Ext: json.RawMessage(`{"prebid":{"type":""}}`),
	}
	result, err := adapter.getMediaType(bid)
	assert.Error(t, err, "Expected error for empty type")
	assert.Equal(t, openrtb_ext.BidType(""), result, "Expected empty result for empty type")
}

// TestGetMediaType_NilExt verifies that getMediaType returns an error and an empty result
// when the bid's extension is nil.
func TestGetMediaType_NilExt(t *testing.T) {
	adapter := &adapter{}
	bid := &openrtb2.Bid{
		Ext: nil,
	}
	result, err := adapter.getMediaType(bid)
	assert.Error(t, err, "Expected error for nil extension")
	assert.Equal(t, openrtb_ext.BidType(""), result, "Expected empty result for nil extension")
}
