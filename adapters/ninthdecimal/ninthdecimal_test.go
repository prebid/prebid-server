package ninthdecimal

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "ninthdecimaltest", NewNinthdecimalBidder("http://rtb.ninthdecimal.com/xp/get?pubid={{.PublisherID}}"))
}
