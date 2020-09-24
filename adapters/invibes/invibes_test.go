package invibes

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "invibestest", NewInvibesBidder("https://adweb.videostepstage.com/bid/VideoAdContent"))
}
