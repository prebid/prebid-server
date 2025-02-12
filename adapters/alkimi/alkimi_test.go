package alkimi

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

const (
	alkimiTestEndpoint = "https://exchange.alkimi-onboarding.com/server/bid"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderAlkimi,
		config.Adapter{Endpoint: alkimiTestEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "alkimitest", bidder)
}

func TestEndpointEmpty(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAlkimi, config.Adapter{
		Endpoint: ""}, config.Server{ExternalUrl: alkimiTestEndpoint, GvlID: 1, DataCenter: "2"})
	assert.Error(t, buildErr)
}

func TestEndpointMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAlkimi, config.Adapter{
		Endpoint: " http://leading.space.is.invalid"}, config.Server{ExternalUrl: alkimiTestEndpoint, GvlID: 1, DataCenter: "2"})
	assert.Error(t, buildErr)
}

func TestBuilder(t *testing.T) {
	bidder, buildErr := buildBidder()
	if buildErr != nil {
		t.Fatalf("Failed to build bidder: %v", buildErr)
	}
	assert.NotNil(t, bidder)
}

func buildBidder() (adapters.Bidder, error) {
	return Builder(
		openrtb_ext.BidderAlkimi,
		config.Adapter{Endpoint: alkimiTestEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"},
	)
}
