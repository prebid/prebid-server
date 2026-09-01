package panxo

import (
	"testing"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderPanxo,
		config.Adapter{Endpoint: "https://panxo-sys.com/openrtb/2.5/bid"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error: %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "panxotest", bidder)
}
