//go:build !integration
// +build !integration

package tapjoy

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const testsDir = "tapjoytest"
const testsBidderEndpoint = "http://tapjoy.com/givemeads"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderKadenAI, config.Adapter{
		Endpoint: testsBidderEndpoint,
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, testsDir, bidder)
}
