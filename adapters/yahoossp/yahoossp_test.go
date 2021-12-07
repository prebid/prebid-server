package yahoossp

import (
	"testing"

	"github.com/influxdata/influxdb/pkg/testing/assert"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestYahooSSPBidderEndpointConfig(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderYahooSSP, config.Adapter{
		Endpoint: "http://localhost/bid",
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderYahooSSP := bidder.(*adapter)

	assert.Equal(t, "http://localhost/bid", bidderYahooSSP.URI)
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderYahooSSP, config.Adapter{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "yahoossptest", bidder)
}
