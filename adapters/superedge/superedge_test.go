package superedge

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderSuperEdge, config.Adapter{
		Endpoint: "https://rtb-us.superedge.co.jp/bid?sk={{.sk}}"}, config.Server{ExternalUrl: "http://hosturl.com"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "superedgetest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderSuperEdge, config.Adapter{Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com"})

	assert.Error(t, buildErr)
}
