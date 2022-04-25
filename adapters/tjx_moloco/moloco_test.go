package moloco

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const testsDir = "molocotest"
const testsBidderEndpoint = "https://bidfnt-us.adsmoloco.com/tapjoy"
const testsBidderEndpointUSEast = "https://bidfnt-us.adsmoloco.com/tapjoy"
const testsBidderEndpointEU = "https://bidfnt-eu.adsmoloco.com/tapjoy"
const testsBidderEndpointAPAC = "https://bidfnt-asia.adsmoloco.com/tapjoy"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMoloco, config.Adapter{
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
