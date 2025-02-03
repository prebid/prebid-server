package roulax

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

const testsDir = "roulaxtest"
const testsBidderEndpoint = "http://dsp.rcoreads.com/api/vidmate?pid=vidmate_android_banner"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderRoulax,
		config.Adapter{Endpoint: testsBidderEndpoint},
		config.Server{ExternalUrl: "http://dsp.rcoreads.com/api/vidmate?pid=vidmate_android_banner", GvlID: 1, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, testsDir, bidder)
}
