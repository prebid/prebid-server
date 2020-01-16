package adtelligent

import (
	"testing"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "adtelligenttest", NewAdtelligentBidder("http://hb.adtelligent.com/auction"))
}
