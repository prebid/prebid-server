package brightroll

import (
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "brightrolltest", NewBrightrollBidder("http://east-bid.ybp.yahoo.com/bid/appnexuspbs"))
}
