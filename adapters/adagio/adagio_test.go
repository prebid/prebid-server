package adagio

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderAdagio,
		config.Adapter{Endpoint: "https://mp-ams.4dex.io/pbserver"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"},
	)

	if err != nil {
		t.Fatalf("Builder returned unexpected error %v", err)
	}

	adapterstest.RunJSONBidderTest(t, "adagiotest", bidder)
}
