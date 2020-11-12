package mobfoxpb

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	mobfoxpbAdapter := NewMobfoxpbBidder("http://example.com/?c=o&m=ortb")
	adapterstest.RunJSONBidderTest(t, "mobfoxpbtest", mobfoxpbAdapter)
}
