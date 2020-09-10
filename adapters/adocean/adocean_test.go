package adocean

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderAdOcean, config.Adapter{
		Endpoint: "https://{{.Host}}"})
	adapterstest.RunJSONBidderTest(t, "adoceantest", bidder)
}
