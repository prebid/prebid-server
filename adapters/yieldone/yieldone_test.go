package yieldone

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "yieldonetest", NewYieldoneBidder("http://localhost/prebid"))
}
