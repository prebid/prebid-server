package blis

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBlis, config.Adapter{
		Endpoint: "https://example.endpoint"},
		config.Server{ExternalUrl: "http://example.server", GvlID: 1, DataCenter: "2"})

	require.NoError(t, buildErr)

	adapterstest.RunJSONBidderTest(t, "blistest", bidder)
}
