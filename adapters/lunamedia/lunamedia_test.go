package lunamedia

import (
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "lunamediatest", NewLunaMediaBidder("http://api.lunamedia.io/xp/get?pubid={{.PublisherID}}"))
}
