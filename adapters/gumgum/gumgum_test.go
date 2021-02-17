package gumgum

import (
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "gumgumtest", NewGumGumBidder("https://g2.gumgum.com/providers/prbds2s/bid"))
}
