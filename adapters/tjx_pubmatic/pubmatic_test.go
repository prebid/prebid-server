//go:build !integration
// +build !integration

package pubmatic

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderPubmatic, config.Adapter{
		Endpoint: "https://hbopenbid.pubmatic.com/translator?source=prebid-server"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "pubmatictest", bidder)
}
