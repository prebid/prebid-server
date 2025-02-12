package vungle

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	conf := config.Adapter{
		Endpoint: "https://vungle.io/bit/t",
	}
	bidder, buildErr := Builder(openrtb_ext.BidderVungle, conf, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 667, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "vungletest", bidder)
}
