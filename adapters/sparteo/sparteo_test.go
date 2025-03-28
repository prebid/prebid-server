package sparteo

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

func int64Ptr(i int64) *int64 {
	return &i
}

// TestBuilder verifies that the Builder function correctly creates a bidder instance.
// It checks for errors, ensures the returned bidder is not nil, and confirms that the endpoint
// in the adapter is set according to the configuration.
func TestBuilder(t *testing.T) {
	cfg := config.Adapter{Endpoint: "https://bid.sparteo.com/s2s-auction"}
	bidder, err := Builder(openrtb_ext.BidderSparteo, cfg, config.Server{GvlID: 1028})

	require.NoError(t, err, "Builder returned an error")
	assert.NotNil(t, bidder, "Bidder is nil")

	sparteoAdapter, ok := bidder.(*adapter)
	require.True(t, ok, "Expected *adapter, got %T", bidder)

	assert.Equal(t, "https://bid.sparteo.com/s2s-auction", sparteoAdapter.endpoint, "Endpoint is not correctly set")
}

// TestJsonSamples runs JSON sample tests using the shared adapterstest framework.
func TestJsonSamples(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{
		Endpoint: "https://bid.sparteo.com/s2s-auction",
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

// TestMakeRequests_PublisherExt_UnmarshalError verifies that when the publisher extension JSON is invalid (i.e. not an object),
// the adapter resets it to an empty map and correctly merges the networkId from the impression bidder extension.
func TestMakeRequests_PublisherExt_UnmarshalError(t *testing.T) {
	bidder, _ := Builder(
		openrtb_ext.BidderSparteo,
		config.Adapter{Endpoint: "https://bid.sparteo.com/s2s-auction"},
		config.Server{},
	)

	imp := openrtb2.Imp{
		ID: "imp1",
		Banner: &openrtb2.Banner{
			W: int64Ptr(300),
			H: int64Ptr(250),
		},
		Ext: json.RawMessage(`{"bidder":{"networkId":"netPub"}}`),
	}

	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp},
		Site: &openrtb2.Site{
			Domain: "test.sparteo.com",
			Publisher: &openrtb2.Publisher{
				Domain: "test.sparteo.com",
				Ext:    json.RawMessage(`"not an object"`),
			},
		},
	}

	requests, _ := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})

	assert.NotNil(t, requests, "Expected a valid request despite publisher ext issue")
	require.NotNil(t, req.Site.Publisher.Ext, "Publisher ext should be set")

	var pubExt map[string]interface{}
	err := json.Unmarshal(req.Site.Publisher.Ext, &pubExt)
	require.NoError(t, err, "Updated publisher ext should unmarshal")
	params, ok := pubExt["params"].(map[string]interface{})
	require.True(t, ok, "Expected publisher ext 'params' to be a map")
	assert.Equal(t, "netPub", params["networkId"], "Expected networkId to be set from imp bidder ext")
}

// TestMakeRequests_PublisherExt_ParamsNotMap verifies that when the publisher extension's "params" field is not a map,
// the adapter replaces it with a new map and correctly merges the networkId from the bidder extension.
func TestMakeRequests_PublisherExt_ParamsNotMap(t *testing.T) {
	bidder, _ := Builder(
		openrtb_ext.BidderSparteo,
		config.Adapter{Endpoint: "https://bid.sparteo.com/s2s-auction"},
		config.Server{},
	)

	imp := openrtb2.Imp{
		ID: "imp1",
		Banner: &openrtb2.Banner{
			W: int64Ptr(300),
			H: int64Ptr(250),
		},
		Ext: json.RawMessage(`{"bidder":{"networkId":"net123"}}`),
	}

	req := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{imp},
		Site: &openrtb2.Site{
			Domain: "test.sparteo.com",
			Publisher: &openrtb2.Publisher{
				Domain: "test.sparteo.com",
				Ext:    json.RawMessage(`{"params": "should be an object"}`),
			},
		},
	}

	requests, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})
	require.Empty(t, errs, "Expected no errors from publisher ext processing")
	require.Len(t, requests, 1, "Expected one request")

	require.NotNil(t, req.Site.Publisher.Ext, "Publisher ext should be set")
	var pubExt map[string]interface{}
	err := json.Unmarshal(req.Site.Publisher.Ext, &pubExt)
	require.NoError(t, err, "Updated publisher ext should unmarshal")
	params, ok := pubExt["params"].(map[string]interface{})
	require.True(t, ok, "Expected publisher ext 'params' to be a map after type correction")
	assert.Equal(t, "net123", params["networkId"], "Expected networkId to be merged into publisher ext")
}
