package consumable

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	clock := knownInstant(time.Date(2016, 1, 1, 12, 30, 15, 0, time.UTC))
	adapterstest.RunJSONBidderTest(t, "consumable", testConsumableBidder(clock, "http://serverbid/api/v2"))
}
