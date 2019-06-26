package advangelists

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "advangeliststest", NewAdvangelistsBidder("http://nep.advangelists.com/xp/get?pubid=19f1b372c7548ec1fe734d2c9f8dc688"))
}
