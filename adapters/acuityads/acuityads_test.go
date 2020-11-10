package acuityads

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "acuityadstest", NewAcuityAdsBidder("http://{{.Host}}.example.com/bid?token={{.AccountID}}"))
}
