package oath

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "oathtest", NewOathBidder("http://east-bid.ybp.yahoo.com/bid/appnexuspbs"))
}
