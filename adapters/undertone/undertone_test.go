package undertone

import (
	"testing"

	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderUndertone, config.Adapter{
		Endpoint: "http://undertone-test/bid",
	},
		config.Server{
			ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2",
		})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "undertonetest", bidder)
}
