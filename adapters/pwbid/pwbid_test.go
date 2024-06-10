package pwbid

import (
	"testing"

	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderPWBid, config.Adapter{
		Endpoint: "https://bidder.east2.pubwise.io/bid/pubwisedirect"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 842, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "pwbidtest", bidder)
}
