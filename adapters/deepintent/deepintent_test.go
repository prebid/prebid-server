package deepintent

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	deepintentAdapter := NewDeepintentBidder("https://prebid.deepintent.com/prebid")
	adapterstest.RunJSONBidderTest(t, "deepintenttest", deepintentAdapter)
}
