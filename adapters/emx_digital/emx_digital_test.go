package emx_digital

import (
	"testing"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderEmxDigital, config.Adapter{
		Endpoint: "https://hb.emxdgt.com"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	setTesting(bidder)
	adapterstest.RunJSONBidderTest(t, "emx_digitaltest", bidder)
}

func setTesting(bidder adapters.Bidder) {
	bidderEmxDigital, _ := bidder.(*EmxDigitalAdapter)
	bidderEmxDigital.testing = true
}
