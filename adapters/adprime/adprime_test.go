package adprime

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderAdprime, config.Adapter{
		Endpoint: "http://delta.adprime.com/?c=o&m=ortb"})
	adapterstest.RunJSONBidderTest(t, "adprimetest", bidder)
}
