package madvertise

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderMadvertise, config.Adapter{
		Endpoint: "{{Malformed}}"})

	assert.Error(t, buildErr)
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMadvertise, config.Adapter{
		Endpoint: "https://mobile.mng-ads.com/bidrequest{{.ZoneID}}"})

	assert.NoError(t, buildErr, "Builder returned unexpected error %v", buildErr)

	adapterstest.RunJSONBidderTest(t, "madvertisetest", bidder)
}
