package admixer

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderAdmixer, config.Adapter{
		Endpoint: "http://inv-nets.admixer.net/pbs.aspx"})
	adapterstest.RunJSONBidderTest(t, "admixertest", bidder)
}
