package blue

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestBidderBlue(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderInMobi, config.Adapter{
		Endpoint: "https://foo.io/?src=prebid"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "bluetest", bidder)
}
