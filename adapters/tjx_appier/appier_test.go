package appier

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const testsDir = "appiertest"

const testsBidderEndpoint = "http://useast.appier.com/givemeads"
const testsBidderEndpointUSEast = "http://useast.appier.com/givemeads"
const testsBidderEndpointEMEA = "http://emea.appier.com/givemeads"
const testsBidderEndpointJP = "http://jp.appier.com/givemeads"
const testsBidderEndpointSG = "http://sg.appier.com/givemeads"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAppier, config.Adapter{
		Endpoint: testsBidderEndpoint,
		XAPI: config.AdapterXAPI{
			EndpointUSEast: testsBidderEndpointUSEast,
			EndpointEMEA:   testsBidderEndpointEMEA,
			EndpointJP:     testsBidderEndpointJP,
			EndpointSG:     testsBidderEndpointSG,
		},
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, testsDir, bidder)
}
