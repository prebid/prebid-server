package smartadserver

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "smartadservertest", NewSmartadserverBidder("https://ssb.smartadserver.com"))
}
