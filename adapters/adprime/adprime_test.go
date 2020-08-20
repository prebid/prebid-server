package adprime

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adprimeAdapter := NewAdprimeBidder("http://delta.adprime.com/?c=o&m=ortb")
	adapterstest.RunJSONBidderTest(t, "adprimetest", adprimeAdapter)
}
