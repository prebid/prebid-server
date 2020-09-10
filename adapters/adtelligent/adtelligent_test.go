package adtelligent

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderAdtelligent, config.Adapter{
		Endpoint: "http://ghb.adtelligent.com/pbs/ortb"})

	adapterstest.RunJSONBidderTest(t, "adtelligenttest", bidder)
}
