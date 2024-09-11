package yahooAds

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestYahooAdsBidderEndpointConfig(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderYahooAds, config.Adapter{
		Endpoint: "http://localhost/bid",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderYahooAds := bidder.(*adapter)

	assert.Equal(t, "http://localhost/bid", bidderYahooAds.URI)
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderYahooAds, config.Adapter{}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "yahooAdstest", bidder)
}
