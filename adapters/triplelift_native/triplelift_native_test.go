package triplelift_native

import (
	"fmt"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBadConfig(t *testing.T) {
	badServer := NewTripleliftNativeBidder(nil, "", "{foo:2}")
	assert.NotEmpty(t, badServer, "NewTripleliftBidder should not return nil")
	expected := &adapters.MisconfiguredBidder{
		Name:  "TripleliftNativeAdapter",
		Error: fmt.Errorf("TripleliftNativeAdapter could not unmarshal config json"),
	}
	assert.IsType(t, expected, badServer, "expected misconfigured server")
}

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "triplelift_nativetest", NewTripleliftNativeBidder(nil, "http://tlx.3lift.net/s2sn/auction?supplier_id=20", "{\"publisher_whitelist\":[\"foo\",\"bar\",\"baz\"]}"))
}
