package ninthdecimal

import (
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "ninthdecimaltest", NewNinthDecimalBidder("http://rtb.ninthdecimal.com/xp/get?pubid={{.PublisherID}}"))
}
