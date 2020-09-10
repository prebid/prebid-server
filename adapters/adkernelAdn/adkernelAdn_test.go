package adkernelAdn

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderAdkernelAdn, config.Adapter{
		Endpoint: "http://{{.Host}}/rtbpub?account={{.PublisherID}}"})
	adapterstest.RunJSONBidderTest(t, "adkerneladntest", bidder)
}
