package mabidder

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMabidder, config.Adapter{
		Endpoint: "https://prebid.ecdrsvc.com/pbs"},
		config.Server{ExternalUrl: "https://prebid.ecdrsvc.com/pbs", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "mabiddertest", bidder)
}
