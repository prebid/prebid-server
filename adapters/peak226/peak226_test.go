package peak226

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderPeak226, config.Adapter{
		Endpoint: "https://{{.Region}}.a.viddea.com/edge_direct"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "peak226test", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderPeak226, config.Adapter{
		Endpoint: "{{Malformed}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestStripBidderExt(t *testing.T) {
	testCases := []struct {
		name        string
		impExt      json.RawMessage
		expectedExt json.RawMessage
		expectError bool
	}{
		{
			name:        "nil ext is left nil",
			impExt:      nil,
			expectedExt: nil,
		},
		{
			name:        "empty ext is cleared",
			impExt:      json.RawMessage(``),
			expectedExt: nil,
		},
		{
			name:        "ext holding only bidder is cleared entirely",
			impExt:      json.RawMessage(`{"bidder":{"publisherId":"pub-1"}}`),
			expectedExt: nil,
		},
		{
			name:        "non-bidder keys are preserved",
			impExt:      json.RawMessage(`{"bidder":{"publisherId":"pub-1"},"gpid":"/1234/home"}`),
			expectedExt: json.RawMessage(`{"gpid":"/1234/home"}`),
		},
		{
			name:        "malformed json returns an error",
			impExt:      json.RawMessage(`{"bidder":`),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			imp := openrtb2.Imp{ID: "imp-1", Ext: tc.impExt}
			err := stripBidderExt(&imp)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tc.expectedExt == nil {
				assert.Nil(t, imp.Ext)
			} else {
				assert.JSONEq(t, string(tc.expectedExt), string(imp.Ext))
			}
		})
	}
}

func TestSetPublisherIDEmpty(t *testing.T) {
	// An empty publisher ID must leave the request untouched rather than creating an
	// empty site/app publisher object.
	request := openrtb2.BidRequest{ID: "req-1"}
	setPublisherID(&request, "")

	assert.Nil(t, request.Site)
	assert.Nil(t, request.App)
}
