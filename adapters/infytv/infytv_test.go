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
		Endpoint: "https://test.infy.tv/pbs/openrtb"})

	assert.NoError(t, buildErr)
	adapterstest.RunJSONBidderTest(t, "infytvtest", bidder)
}
