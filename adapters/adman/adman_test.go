package adman

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderAppnexus, config.Adapter{
		Endpoint: "http://pub.admanmedia.com/?c=o&m=ortb"})
	adapterstest.RunJSONBidderTest(t, "admantest", bidder)
}
