package ttx

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.Bidder33Across, config.Adapter{
		Endpoint: "http://ssc.33across.com"})
	adapterstest.RunJSONBidderTest(t, "33across", bidder)
}
