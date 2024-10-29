package loopme

import (
	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderLoopme,
		config.Adapter{
			Endpoint: "http://prebid.loopmertb.com",
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       109,
			DataCenter:  "2",
		},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "loopmetest", bidder)
}
