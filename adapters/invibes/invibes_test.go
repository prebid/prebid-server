package invibes

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderInvibes, config.Adapter{
		Endpoint: "https://adweb.videostepstage.com/bid/VideoAdContent"})

	if buildErr != nil {
		t.Fatalf("Builder returned expected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "invibestest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAdhese, config.Adapter{
		Endpoint: "{{Malformed}}"})

	assert.Error(t, buildErr)
}
