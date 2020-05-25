package smaato

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "smaatotest", NewSmaatoBidder(nil, "https://prebid-test.smaatolabs.net/bidder"))
}
