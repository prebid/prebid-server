package riptide

import (
	"testing"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// TestJsonSamples runs upstream's JSON bidder test over riptidetest/: every case is a mock bid
// request, the exact HTTP calls the adapter must make and the typed bids (or errors) it returns.
func TestJsonSamples(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderRiptide, config.Adapter{Endpoint: "https://serve.riptideads.com/ortb"}, config.Server{})
	if err != nil {
		t.Fatalf("Builder: %v", err)
	}
	adapterstest.RunJSONBidderTest(t, "riptidetest", bidder)
}

func TestBuilderRejectsAnEndpointThatIsNotALiteralAbsoluteURL(t *testing.T) {
	for _, endpoint := range []string{"", "serve.riptideads.com/ortb", "/ortb", "https://{{.Host}}/ortb", "https://%zz/ortb"} {
		if _, err := Builder(openrtb_ext.BidderRiptide, config.Adapter{Endpoint: endpoint}, config.Server{}); err == nil {
			t.Errorf("Builder accepted endpoint %q", endpoint)
		}
	}
}
