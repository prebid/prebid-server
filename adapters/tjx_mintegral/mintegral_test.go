package mintegral

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const testsDir = "mintegraltest"
const testsBidderEndpoint = "http://sg.mintegral.com/givemeads"
const testsBidderEndpointSG = "http://sg.mintegral.com/givemeads"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMintegral, config.Adapter{
		Endpoint: testsBidderEndpoint,
		XAPI: config.AdapterXAPI{
			EndpointSG: testsBidderEndpointSG,
		},
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, testsDir, bidder)
}
