package aduptech

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdUpTech, config.Adapter{
		Endpoint: "https://example.com/rtb/bid", ExtraAdapterInfo: "{\"target_currency\": \"EUR\"}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.NoError(t, buildErr, "Builder returned unexpected error")

	adapterstest.RunJSONBidderTest(t, "aduptechtest", bidder)
}
