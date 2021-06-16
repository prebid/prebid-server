package salunamedia

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderSaLunaMedia, config.Adapter{
		Endpoint: "http://test.com/pserver"})

	assert.NoError(t, buildErr)
	adapterstest.RunJSONBidderTest(t, "salunamediatest", bidder)
}
