package displayio

import (
	"testing"

	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderDisplayio,
		config.Adapter{Endpoint: "https://adapter.endpoint/?macro={{.PublisherID}}"},
		config.Server{ExternalUrl: "https://server.endpoint/"},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "displayiotest", bidder)
}
