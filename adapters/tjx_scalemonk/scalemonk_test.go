package scalemonk

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const testsDir = "scalemonktest"
const testsBidderEndpoint = "http://scalemonk.com/givemeads"
const testsBidderEndpointUSEast = "http://scalemonk-us-east.com/givemeads"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderScaleMonk, config.Adapter{
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
