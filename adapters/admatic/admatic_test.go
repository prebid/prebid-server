package admatic

import (
	"testing"

	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdmatic, config.Adapter{
		Endpoint: "http://pbs.admatic.com.tr?host={{.Host}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1281, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "admatictest", bidder)
}
