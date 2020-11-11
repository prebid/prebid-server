package ttx

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "33acrosstest", New33AcrossBidder("http://ssc.33across.com"))
}
