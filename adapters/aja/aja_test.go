package aja

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const testsBidderEndpoint = "https://localhost/bid/4"

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderAJA, config.Adapter{
		Endpoint: testsBidderEndpoint})
	adapterstest.RunJSONBidderTest(t, "ajatest", bidder)
}
