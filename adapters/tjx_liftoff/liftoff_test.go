package liftoff

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const testsDir = "liftofftest"
const testsBidderEndpoint = "http://liftoff.com/givemeads"
const testsBidderEndpointUSEast = "http://liftoff-us-east.com/givemeads"
const testsBidderEndpointEU = "http://liftoff-eu.com/givemeads"
const testsBidderEndpointAPAC = "http://liftoff-apac.com/givemeads"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderLiftoff, config.Adapter{
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
