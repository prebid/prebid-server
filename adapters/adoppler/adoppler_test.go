package adoppler

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderAdoppler, config.Adapter{
		Endpoint: "http://adoppler.com"})
	adapterstest.RunJSONBidderTest(t, "adopplertest", bidder)
}
