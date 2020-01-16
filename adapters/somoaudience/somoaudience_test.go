package somoaudience

import (
	"testing"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "somoaudiencetest", NewSomoaudienceBidder("http://publisher-east.mobileadtrading.com/rtb/bid"))
}
