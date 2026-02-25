package revx_test

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/adapters/revx"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := revx.Builder(openrtb_ext.BidderRevX, config.Adapter{
		Endpoint: "prebid-use.atomex.net/ag=PUB123",
	}, config.Server{
		ExternalUrl: "http://hosturl.com",
		GvlID:       375,
		DataCenter:  "2",
	})

	if buildErr != nil {

		t.Logf("RevX Builder created successfully: %+v", bidder)
	}

	adapterstest.RunJSONBidderTest(t, "revxtest", bidder)
}
