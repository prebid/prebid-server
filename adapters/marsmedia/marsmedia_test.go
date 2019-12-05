package marsmedia

import (
	"testing"

	"github.com/prebid/prebid-server/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "marsmediatest", NewMarsmediaBidder("http://bid306.rtbsrv.com/bidder/?bid=f3xtet&dev=1"))
}
