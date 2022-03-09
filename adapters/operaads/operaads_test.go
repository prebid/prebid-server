package operaads

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const testsDir = "operaadstest"

const testsBidderEndpoint = "http://operaads.com/givemeads"
const testsBidderEndpointUSEast = "http://operaads.com/givemeads_useast"
const testsBidderEndpointAPAC = "http://operaads.com/givemeads_apac"
const testsBidderEndpointEU = "http://operaads.com/givemeads_eu"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderOperaAds, config.Adapter{
		Endpoint: testsBidderEndpoint,
		XAPI: config.AdapterXAPI{
			EndpointUSEast: testsBidderEndpointUSEast,
			EndpointAPAC:   testsBidderEndpointAPAC,
			EndpointEU:     testsBidderEndpointEU,
		},
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, testsDir, bidder)
}
