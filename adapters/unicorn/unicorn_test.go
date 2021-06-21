package unicorn

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	// adapterstest.RunJSONBidderTest(t, testsDir, NewUnicornBidder(http.DefaultClient, "https://jp.unicorn.com/tapjoy", "https://jp.unicorn.com/tapjoy"))
	bidder, buildErr := Builder(openrtb_ext.BidderUnicorn, config.Adapter{
		Endpoint: "https://ds.uncn.jp"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "unicorntest", bidder)
}
