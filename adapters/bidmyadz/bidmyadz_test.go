package bidmyadz

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBidmyadz, config.Adapter{
		Endpoint: "http://endpoint.bidmyadz.com/c0f68227d14ed938c6c49f3967cbe9bc"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.NoError(t, buildErr)
	adapterstest.RunJSONBidderTest(t, "bidmyadztest", bidder)
}
