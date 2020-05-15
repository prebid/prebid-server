package mobilefuse

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "mobilefusetest", NewMobilefuseBidder("https://{{.Host}}/prebid/bid"))
}
