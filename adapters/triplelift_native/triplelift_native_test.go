package triplelift_native

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "triplelifttest", NewTripleliftNativeBidder(nil, "http://tlx.3lift.net/s2s/auction?supplier_id=19", "{\"publisher_whitelist\":[]}"))
}
