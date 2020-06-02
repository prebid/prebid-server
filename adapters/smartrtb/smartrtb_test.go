package smartrtb

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "smartrtbtest", NewSmartRTBBidder("http://market-east.smrtb.com/json/publisher/rtb?pubid=test"))
}
