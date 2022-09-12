package infytv

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderEVolution, config.Adapter{
		Endpoint: "https://test.infy.tv/pbs/openrtb"}, config.Server{ExternalUrl: "http://hosturl.com", GdprID: 1, Datacenter: "2"})

	assert.NoError(t, buildErr)
	adapterstest.RunJSONBidderTest(t, "infytvtest", bidder)
}
