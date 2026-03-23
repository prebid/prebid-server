package nativo

import (
	"testing"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestBidderNativo(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderNativo, config.Adapter{
		Endpoint: "https://foo.io/?src=prebid"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "nativotest", bidder)
}
