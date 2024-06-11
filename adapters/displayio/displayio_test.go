package displayio

import (
	"testing"

	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"

	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderDisplayio,
		config.Adapter{Endpoint: "https://prebid.display.io/?publisher={{.PublisherID}}"},
		config.Server{ExternalUrl: "https://prebid.display.io/?publisher=101"},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "displayiotest", bidder)
}
