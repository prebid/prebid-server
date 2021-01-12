package adoppler

import (
	"testing"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	bidder := NewAdopplerBidder("http://{{.AccountID}}.trustedmarketplace.com/processHeaderBid/{{.AdUnit}}")
	adapterstest.RunJSONBidderTest(t, "adopplertest", bidder)
}
