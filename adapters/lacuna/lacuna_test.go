package lacuna

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderLacuna, config.Adapter{
		Endpoint: "http://test-bid.lacunads.com/v1/prebid"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "lacunatest", bidder)
}

func TestJsonSamples2(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderLacuna, config.Adapter{
		Endpoint: "http://test-bid.lacunads.com/v1/prebid"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 0, DataCenter: ""})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "lacunatest", bidder)
}
