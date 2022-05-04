package dv360

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const testsDir = "dv360test"
const testsBidderEndpoint = "https://bid.g.doubleclick.net/xbbe/bid/tapjoy"
const testsBidderEndpointUSEast = "https://bid.g.doubleclick.net/xbbe/bid/tapjoy"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderDV360, config.Adapter{
		Endpoint: testsBidderEndpoint,
		XAPI: config.AdapterXAPI{
			EndpointUSEast: testsBidderEndpointUSEast,
		},
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, testsDir, bidder)
}
