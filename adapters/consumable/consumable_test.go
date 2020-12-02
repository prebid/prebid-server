package consumable

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderConsumable, config.Adapter{
		Endpoint: "http://ib.adnxs.com/openrtb2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	setKnownTime(bidder)
	adapterstest.RunJSONBidderTest(t, "consumable", bidder)
}

func setKnownTime(bidder adapters.Bidder) {
	bidderConsumable, _ := bidder.(*ConsumableAdapter)
	bidderConsumable.clock = knownInstant(time.Date(2016, 1, 1, 12, 30, 15, 0, time.UTC))
}
