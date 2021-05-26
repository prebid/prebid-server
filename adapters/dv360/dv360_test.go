package dv360

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	conf := config.Adapter{
		Endpoint: "https://bid.g.doubleclick.net/xbbe/bid/tapjoy",
	}
	bidder, buildErr := Builder(openrtb_ext.BidderPangle, conf)
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "pangletest", bidder)
}
