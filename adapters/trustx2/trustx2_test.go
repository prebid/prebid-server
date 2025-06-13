package trustx2

import (
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTrustX2, config.Adapter{
		Endpoint: "https://test.localhost.com"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	require.NoError(t, buildErr, "Builder returned unexpected error")
	adapterstest.RunJSONBidderTest(t, "trustx2test", bidder)
}
