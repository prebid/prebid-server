package showheroes

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderShowheroes, config.Adapter{
		Endpoint: "https://bid.showheroes.com/api/v1/bid",
	}, config.Server{})
	if err != nil {
		t.Fatalf("Builder returned unexpected error %v", err)
	}

	adapterstest.RunJSONBidderTest(t, "showheroestest", bidder)
}
