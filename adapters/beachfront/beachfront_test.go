package beachfront

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "beachfronttest", NewBeachfrontBidder("https://display.bfmio.com/prebid_display", "{\"video_endpoint\":\"https://reachms.bfmio.com/bid.json?exchange_id\"}"))
}
