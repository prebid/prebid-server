package silvermob

import (
	"testing"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "silvermobtest", NewSilverMobBidder("http://{{.Host}}.example.com/api/dsp/bid/{{.ZoneID}}"))
}
