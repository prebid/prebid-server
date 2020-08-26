package brightroll

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "brightrolltest", NewBrightrollBidder("http://test-bid.ybp.yahoo.com/bid/appnexuspbs", "{\"account\": [{\"id\": \"adthrive\",\"badv\": [], \"bcat\": [\"IAB8-5\",\"IAB8-18\"],\"battr\": [1,2,3], \"bidfloor\":0.0}]}"))
}
