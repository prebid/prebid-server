package criteo

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {

	// Setup:
	bidder, buildErr := builderWithGuidGenerator(
		openrtb_ext.BidderCriteo,
		config.Adapter{
			Endpoint: "https://bidder.criteo.com/cdb?profileId=230",
		},
		newFakeGuidGenerator("00000000-0000-0000-00000000"),
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Execute & Verify:
	adapterstest.RunJSONBidderTest(t, "criteotest", bidder)
}
