package adhese

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderAdhese, config.Adapter{
		Endpoint: "https://ads-{{.AccountID}}.adhese.com/json"})
	adapterstest.RunJSONBidderTest(t, "adhesetest", bidder)
}
