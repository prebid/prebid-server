package adtelligent

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "adtelligenttest", NewAdtelligentBidder("http://ghb.adtelligent.com/pbs/ortb"))
}
