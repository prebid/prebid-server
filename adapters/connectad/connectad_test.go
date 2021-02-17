package connectad

import (
	"testing"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "connectadtest", NewConnectAdBidder(("http://bidder.connectad.io/API?src=pbs")))
}
