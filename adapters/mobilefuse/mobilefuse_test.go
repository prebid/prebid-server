package mobilefuse

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMobileFuse, config.Adapter{
		Endpoint: "http://mfx.mobilefuse.com/openrtb?pub_id={{.PublisherID}}"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "mobilefusetest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderMobileFuse, config.Adapter{
		Endpoint: "{{Malformed}}"})

	assert.Error(t, buildErr)
}
