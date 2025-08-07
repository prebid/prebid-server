package goldbach

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	_, buildErr := Builder(
		openrtb_ext.BidderGoldbach,
		config.Adapter{},
		config.Server{},
	)

	if buildErr == nil {
		t.Fatalf("Builder should have returned error")
	}

	bidder, buildErr := Builder(
		openrtb_ext.BidderGoldbach,
		config.Adapter{
			Endpoint: "https://gold.bach/prebid/adapter-endpoint",
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       1,
			DataCenter:  "2",
		},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "goldbachtest", bidder)
}
