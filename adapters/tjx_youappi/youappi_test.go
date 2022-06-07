package youappi

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderYouAppi, config.Adapter{
		Endpoint: "https://tapjoy.youappi.net/rtb/bid",
		XAPI: config.AdapterXAPI{
			EndpointEU:     "https://tapjoyeu.youappi.net/rtb/bid",
			EndpointAPAC:   "https://tapjoyapac.youappi.net/rtb/bid",
			EndpointUSEast: "https://tapjoy.youappi.net/rtb/bid",
		},
	})

	if err != nil {
		t.Fatalf("Builder returned unexpected error %v", err)
	}

	adapterstest.RunJSONBidderTest(t, "youappitest", bidder)
}
