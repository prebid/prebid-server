package aps

import (
	"testing"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderAps,
		config.Adapter{
			Endpoint: "https://s2s.prebid.bid-{{.Region}}.ads.aps.amazon-adsystem.com/e/pb/bid",
		},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 793, DataCenter: "2"},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "apstest", bidder)
}
