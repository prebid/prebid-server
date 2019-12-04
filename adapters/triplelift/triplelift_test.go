package triplelift

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "triplelifttest", NewTripleliftBidder(nil, "http://tlx.3lift.net/s2s/auction?supplier_id=20"))
}
