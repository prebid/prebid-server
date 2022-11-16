package yeahmobi

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const testsDir = "yeahmobitest"
const testsBidderEndpoint = "https://bid.yeahtargeter.com/tapjoy/bid"
const testsBidderEndpointUSEast = "https://bid.yeahtargeter.com/tapjoy/bid"
const testsBidderEndpointUSWest = "https://bid.yeahtargeter.com/tapjoy/bid"
const testsBidderEndpointEU = "https://bid.yeahtargeter.com/tapjoy/bid"
const testsBidderEndpointAPAC = "https://bid.yeahtargeter.com/tapjoy/bid"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderYeahmobi, config.Adapter{
		Endpoint: testsBidderEndpoint,
		XAPI: config.AdapterXAPI{
			EndpointUSEast: testsBidderEndpointUSEast,
			EndpointUSWest: testsBidderEndpointUSWest,
			EndpointAPAC:   testsBidderEndpointAPAC,
			EndpointEU:     testsBidderEndpointEU,
		},
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, testsDir, bidder)
}
