package adpone

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const testsDir = "adponetest"
const testsBidderEndpoint = "http://localhost/prebid_server"

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderAdpone, config.Adapter{
		Endpoint: testsBidderEndpoint})
	adapterstest.RunJSONBidderTest(t, testsDir, bidder)
}
