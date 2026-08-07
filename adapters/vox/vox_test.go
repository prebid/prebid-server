package vox

import (
	"testing"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderVox, config.Adapter{
		Endpoint: "http://somecoolurlfor.vox"},
		config.Server{ExternalUrl: "http://somecoolurlfor.vox", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error: %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "voxtest", bidder)
}
