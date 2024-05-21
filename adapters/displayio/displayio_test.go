package displayio

import (
	"testing"

	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"

	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderDisplayio,
		config.Adapter{Endpoint: "https://101.prebid.display.io"},
		config.Server{ExternalUrl: "https://101.prebid.display.io"},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "displayiotest", bidder)
}
