package aax

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAax, config.Adapter{
		Endpoint:         "https://example.aax.media/rtb/prebid",
		ExtraAdapterInfo: "http://localhost:8080/extrnal_url",
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "aaxtest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAax, config.Adapter{
		Endpoint: "{{Malformed}}"})

	assert.Nil(t, buildErr)
}
