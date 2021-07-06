package bidmyadz

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBidmyadz, config.Adapter{
		Endpoint: "http://endpoint.bidmyadz.com/c0f68227d14ed938c6c49f3967cbe9bc"})

	assert.NoError(t, buildErr)
	adapterstest.RunJSONBidderTest(t, "bidmyadztest", bidder)
}
