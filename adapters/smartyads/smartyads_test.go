package smartyads

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "smartyadstest", NewSmartyAdsBidder("http://{{.Host}}.example.com/bid?rtb_seat_id={{.SourceId}}&secret_key={{.AccountID}}"))
}
