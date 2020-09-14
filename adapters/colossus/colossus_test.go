package colossus

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
<<<<<<< HEAD
	colossusAdapter := NewColossusBidder("http://colossusssp.com/?c=o&m=rtb")
=======
	colossusAdapter := NewColossusBidder("http://example.com/?c=o&m=rtb")
>>>>>>> cd364bae287009a18923abfd943aaee06f03cdb2
	adapterstest.RunJSONBidderTest(t, "colossustest", colossusAdapter)
}
