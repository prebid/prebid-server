package nanointeractive

import (
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "nanointeractivetest", NewNanoIneractiveBidder("https://ad.audiencemanager.de/hbs"))
}
