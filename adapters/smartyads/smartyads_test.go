package smartyads

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "smartyadstest", NewSmartyadsBidder("http://n1.smartyads.com/prebid"))
}
