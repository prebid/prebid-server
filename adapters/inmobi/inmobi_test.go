package inmobi

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "inmobitest", NewInmobiAdapter("https://api.w.inmobi.com/showad/openrtb/bidder/prebid"))
}
