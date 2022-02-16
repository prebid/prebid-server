package rtbhouse

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const testsDir = "rtbhousetest"
const testsBidderEndpoint = "http://rtbhouse.com/givemeads"
const testsBidderEndpointUSEast = "http://rtbhouse.com/givemeads_useast"
const testsBidderEndpointAPAC = "http://rtbhouse.com/givemeads_apac"
const testsBidderEndpointEU = "http://rtbhouse.com/givemeads_eu"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderRTBHouse, config.Adapter{
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
