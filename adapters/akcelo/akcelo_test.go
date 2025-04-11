package akcelo

import (
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderAJA,
		config.Adapter{Endpoint: "https://localhost/bid/4"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"},
	)

	require.NoError(t, buildErr, "Builder returned unexpected error")
	adapterstest.RunJSONBidderTest(t, "akcelotest", bidder)
}
