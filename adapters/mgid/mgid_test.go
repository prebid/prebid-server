package mgid

import (
	"testing"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "mgidtest", NewMgidBidder("https://prebid.mgid.com/prebid/"))
}
