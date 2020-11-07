package ix

import (
	"flag"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	// increase verbosity level to test debug logging
	value := flag.Lookup("v").Value
	level := value.String()
	value.Set("3")
	defer value.Set(level)

	if bidder, err := Builder(openrtb_ext.BidderIx,
		config.Adapter{Endpoint: "http://ib.adnxs.com/openrtb2"}); err == nil {
		maxRequests = 2
		adapterstest.RunJSONBidderTest(t, "ixtest", bidder)
	} else {
		t.Fatalf("Builder returned unexpected error %v", err)
	}
}
