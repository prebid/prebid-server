package ninthdecimal

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderNinthDecimal, config.Adapter{
		Endpoint: "http://rtb.ninthdecimal.com/xp/get?pubid={{.PublisherID}}"}, config.Server{ExternalUrl: "http://hosturl.com", GdprID: 1, Datacenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "ninthdecimaltest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderNinthDecimal, config.Adapter{
		Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com", GdprID: 1, Datacenter: "2"})

	assert.Error(t, buildErr)
}
