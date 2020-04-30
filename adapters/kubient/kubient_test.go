package kubient

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "kubienttest", NewKubientBidder("http://127.0.0.1:5000/bid"))
}
