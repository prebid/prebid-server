package risemediatech

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderRiseMediaTech,
		config.Adapter{
			Endpoint: "https://dev-ads.risemediatech.com/ads/rtb/prebid/server",
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       0,
			DataCenter:  "2",
		},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error: %v", buildErr)
	}
	require.NoError(t, buildErr, "Builder returned unexpected error")

	adapterstest.RunJSONBidderTest(t, "risemediatechtest", bidder)
}
