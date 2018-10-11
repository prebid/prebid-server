package rhythmone

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "rhythmonetest", NewRhythmoneBidder("http://tag.1rx.io/rmp"))
}
