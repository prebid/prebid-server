package brightroll

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "brightrolltest", NewBrightrollBidder("http://test-bid.ybp.yahoo.com/bid/appnexuspbs"))
}
