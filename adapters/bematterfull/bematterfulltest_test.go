package bematterfull

import (
	"testing"

	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderSmartHub,
		config.Adapter{
			Endpoint: "http://prebid-srv.mtflll-system.live/?pid={{.SourceId}}&host={{.Host}}",
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       1,
			DataCenter:  "2",
		},
	)

	assert.NoError(t, buildErr)
	adapterstest.RunJSONBidderTest(t, "bematterfulltest", bidder)
}
