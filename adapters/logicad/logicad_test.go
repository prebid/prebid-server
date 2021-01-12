package logicad

import (
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "logicadtest", NewLogicadBidder("https://localhost/adrequest/prebidserver"))
}
