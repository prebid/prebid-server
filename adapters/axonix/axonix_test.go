package axonix

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamplesWithConfiguredURI(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAxonix, config.Adapter{
		Endpoint: "https://openrtb-us-east-1.axonix.com/supply/prebid-server/24cc9034-f861-47b8-a6a8-b7e0968c00b8"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "axonixtest", bidder)
}

func TestJsonSamplesWithHardcodedURI(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAxonix, config.Adapter{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "axonixtest", bidder)
}
