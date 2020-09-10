package adkernel

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderAdkernel, config.Adapter{
		Endpoint: "http://{{.Host}}/hb?zone={{.ZoneID}}"})
	adapterstest.RunJSONBidderTest(t, "adkerneltest", bidder)
}
