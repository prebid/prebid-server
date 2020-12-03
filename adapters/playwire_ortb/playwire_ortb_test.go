package playwire_ortb

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "gumgumtest", NewOrtbBidder("https://g2.gumgum.com/providers/prbds2s/bid"))
}
