package logicad

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "logicadtest", NewLogicadBidder("https://pbs.ladsp.com/adrequest/prebidserver"))
}
