package blis

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/require"
)

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderBlis, config.Adapter{
		Endpoint: "{{Malformed}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBlis, config.Adapter{
		Endpoint: "https://example.endpoint/{{.SupplyId}}"},
		config.Server{ExternalUrl: "http://example.server", GvlID: 1, DataCenter: "2"})

	require.NoError(t, buildErr)

	adapterstest.RunJSONBidderTest(t, "blistest", bidder)
}
