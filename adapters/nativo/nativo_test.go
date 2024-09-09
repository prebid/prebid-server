package nativo

import (
	"testing"

	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

func TestBidderNativo(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderNativo, config.Adapter{
		Endpoint: "https://exchange.postrelease.com/esi?ntv_epid=7"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "nativotest", bidder)
}
