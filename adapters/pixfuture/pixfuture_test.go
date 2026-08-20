package pixfuture

import (
	"testing"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderPixfuture, config.Adapter{
		Endpoint: "http://any.url",
	}, config.Server{})
	require.NoError(t, buildErr, "Builder returned unexpected error")

	adapterstest.RunJSONBidderTest(t, "pixfuturetest", bidder)
}
