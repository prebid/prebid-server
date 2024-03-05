package eplanning

import (
	"testing"

	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderEPlanning, config.Adapter{
		Endpoint: "http://rtb.e-planning.net/pbs/1"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	setTesting(bidder)
	adapterstest.RunJSONBidderTest(t, "eplanningtest", bidder)
}

func setTesting(bidder adapters.Bidder) {
	bidderEplanning := bidder.(*EPlanningAdapter)
	bidderEplanning.testing = true
}
