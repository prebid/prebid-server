package sharethrough

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/version"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	version.Ver = "42.42"
	strVersion = "10.0"

	bidder, buildErr := Builder(openrtb_ext.BidderSharethrough, config.Adapter{
		Endpoint: "http://whatever.url",
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "sharethroughtest", bidder)
}
