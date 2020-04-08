package beintoo

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "beintootest", NewBeintooBidder("http://localhost/prebid"))
	adapterstest.RunJSONBidderTest(t, "beintootest", NewBeintooBidder(""))
}
