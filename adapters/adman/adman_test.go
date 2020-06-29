package adman

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	admanAdapter := NewAdmanBidder("http://pub.admanmedia.com/?c=o&m=ortb")
	adapterstest.RunJSONBidderTest(t, "admantest", admanAdapter)
}
