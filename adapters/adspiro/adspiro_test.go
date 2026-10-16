package adspiro

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testEndpoint = "https://adspiro.example/pbs?pid={{.PublisherID}}"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdspiro, config.Adapter{
		Endpoint: testEndpoint,
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "adspirotest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAdspiro, config.Adapter{
		Endpoint: "{{Malformed}}",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestMakeRequestsEndpointResolveError(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdspiro, config.Adapter{
		Endpoint: "https://adspiro.example/pbs?pid={{.Unknown}}",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	require.NoError(t, buildErr)

	request := &openrtb2.BidRequest{
		ID: "request-1",
		Imp: []openrtb2.Imp{{
			ID:     "imp-1",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
			Ext:    json.RawMessage(`{"bidder":{"publisherId":"pub-1"}}`),
		}},
	}

	requests, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Nil(t, requests)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "can't evaluate field Unknown")
}
