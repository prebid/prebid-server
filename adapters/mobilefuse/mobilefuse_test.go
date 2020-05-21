package mobilefuse

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "mobilefusetest", NewMobileFuseBidder("http://mfx-us-east.mobilefuse.com/openrtb?pub_id={{.PublisherID}}"))
}
