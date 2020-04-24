package beintoo

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	beintooAdapter := NewBeintooBidder("https://ib.beintoo.com")
	beintooAdapter.testing = true
	adapterstest.RunJSONBidderTest(t, "beintootest", beintooAdapter)
}
