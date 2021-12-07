package interactiveoffers

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderInteractiveoffers, config.Adapter{
		Endpoint: "https://prebid-server.ioadx.com/bidRequest/?partnerId=d9e56d418c4825d466ee96c7a31bf1da6b62fa04"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "interactiveofferstest", bidder)
}
