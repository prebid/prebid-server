package telaria

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestEndpointFromConfig(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTelaria, config.Adapter{
		Endpoint: "providedurl.com",
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderTelari := bidder.(*TelariaAdapter)

	assert.Equal(t, "providedurl.com", bidderTelari.URI)
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTelaria, config.Adapter{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "telariatest", bidder)
}
