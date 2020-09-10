package advangelists

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderAdvangelists, config.Adapter{
		Endpoint: "http://nep.advangelists.com/xp/get?pubid={{.PublisherID}}"})
	adapterstest.RunJSONBidderTest(t, "advangeliststest", bidder)
}
