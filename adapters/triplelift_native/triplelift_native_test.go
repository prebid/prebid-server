package triplelift

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
<<<<<<< HEAD:adapters/triplelift_native/triplelift_native_test.go
	adapterstest.RunJSONBidderTest(t, "triplelifttest", NewTripleliftNativeBidder(nil, "http://tlx.3lift.net/s2s/auction?supplier_id=19"))
=======
	adapterstest.RunJSONBidderTest(t, "triplelifttest", NewTripleliftBidder(nil, "http://tlx.3lift.net/s2s/auction?supplier_id=20"))
>>>>>>> master:adapters/triplelift/triplelift_test.go
}
