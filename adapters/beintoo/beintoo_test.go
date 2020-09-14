package beintoo

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderAppnexus, config.Adapter{
		Endpoint: "https://ib.beintoo.com"})

	adapterstest.RunJSONBidderTest(t, "beintootest", bidder)
}
