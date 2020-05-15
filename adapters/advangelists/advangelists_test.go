package advangelists

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "advangeliststest", NewAdvangelistsBidder("http://nep.advangelists.com/xp/get?pubid={{.PublisherID}}"))
}
