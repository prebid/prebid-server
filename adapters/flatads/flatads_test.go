package flatads

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderFlatads, config.Adapter{
		Endpoint: "https://test.endpoint.com/api/rtbs/adx/rtb?x-net-id={{.PublisherID}}&x-net-token={{.TokenID}}"},
		config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "flatadstest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderFlatads, config.Adapter{
		Endpoint: "x-net-id={{PublisherID}}&x-net-token={{TokenID}}"}, config.Server{})

	assert.Error(t, buildErr)
}
