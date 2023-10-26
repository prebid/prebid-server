package smartx

import (
	"testing"

	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

const testsDir = "smartxtest"
const testsBidderEndpoint = "https://bid.smartclip.net/bid/1005"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderRise,
		config.Adapter{Endpoint: testsBidderEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 115, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, testsDir, bidder)
}
