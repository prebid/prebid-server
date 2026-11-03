package tadvertising_blis

import (
	"testing"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTAdvertisingBlis, config.Adapter{
		Endpoint: "https://example.endpoint/"},
		config.Server{ExternalUrl: "http://example.server", GvlID: 1, DataCenter: "2"})

	require.NoError(t, buildErr)

	adapterstest.RunJSONBidderTest(t, "tadvertising_blistest", bidder)
}
