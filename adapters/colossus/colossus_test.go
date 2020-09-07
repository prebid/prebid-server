package colossus

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	colossusAdapter := NewColossusBidder("http://example.com/?c=o&m=rtb")
	adapterstest.RunJSONBidderTest(t, "colossustest", colossusAdapter)
}
