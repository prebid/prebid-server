package applogy

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderApplogy, config.Adapter{
		Endpoint: "http://example.com/prebid"})
	adapterstest.RunJSONBidderTest(t, "applogytest", bidder)
}
