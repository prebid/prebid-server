package nanointeractive

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "nanointeractivetest", NewNanoIneractiveBidder("https://ad.audiencemanager.de/hbs"))
}
