package liftoff

import (
	"testing"

	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	conf := config.Adapter{
		Endpoint: "https://liftoff.io/bit/t",
	}
	bidder, buildErr := Builder(openrtb_ext.BidderLiftoff, conf, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 667, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "liftofftest", bidder)
}
