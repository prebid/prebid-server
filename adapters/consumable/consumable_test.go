package consumable

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderConsumable, config.Adapter{
		Endpoint: "http://ib.adnxs.com/openrtb2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	assertClock(t, bidder)
	replaceClockWithKnownTime(bidder)

	adapterstest.RunJSONBidderTest(t, "consumable", bidder)
}

func assertClock(t *testing.T, bidder adapters.Bidder) {
	bidderConsumable, _ := bidder.(*ConsumableAdapter)
	assert.NotNil(t, bidderConsumable.clock)
}

func replaceClockWithKnownTime(bidder adapters.Bidder) {
	bidderConsumable, _ := bidder.(*ConsumableAdapter)
	bidderConsumable.clock = knownInstant(time.Date(2016, 1, 1, 12, 30, 15, 0, time.UTC))
}
