package iqzone

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "iqzonetest", NewIQZoneBidder("http://nep.advangelists.com/xp/get?pubid={{.PublisherID}}"))
}
