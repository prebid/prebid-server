package ix

import (
	"flag"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	// increase verbosity level to test debug logging
	value := flag.Lookup("v").Value
	level := value.String()
	value.Set("3")
	defer value.Set(level)

	maxRequests = 2
	adapterstest.RunJSONBidderTest(t, "ixtest", NewIxBidder(nil, "http://ib.adnxs.com/openrtb2"))
}
