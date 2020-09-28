package conversant

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdtelligent, config.Adapter{})

	if buildErr != nil {
		t.Fatalf("Builder returned expected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "conversanttest", bidder)
}
