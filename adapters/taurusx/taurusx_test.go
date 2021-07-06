package taurusx

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTaurusX, config.Adapter{
		Endpoint: "https://useast.taurusx.com/tapjoy",
		XAPI: config.AdapterXAPI{
			EndpointUSEast: "https://useast.taurusx.com/tapjoy",
			EndpointJP:     "https://jp.taurusx.com/tapjoy",
			EndpointSG:     "https://sg.taurusx.com/tapjoy",
		}})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "taurusxtest", bidder)
}
