package exco

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	adapterConfig := config.Adapter{
		Endpoint: "https://testjsonsample.com",
	}
	serverConfig := config.Server{
		ExternalUrl: "http://hosturl.com",
		GvlID:       1,
		DataCenter:  "2",
	}
	bidder, buildErr := Builder(openrtb_ext.BidderExco, adapterConfig, serverConfig)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "excotest", bidder)
}
