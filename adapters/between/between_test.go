package between

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "betweentest", NewBetweenBidder("http://{{.Host}}/{{.PublisherID}}"))
}
