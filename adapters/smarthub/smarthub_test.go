package smarthub

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderSmartHub, config.Adapter{
		Endpoint: "http://{{.Host}}-prebid.smart-hub.io/?seat={{.AccountID}}&token={{.SourceId}}"})

	assert.NoError(t, buildErr)
	adapterstest.RunJSONBidderTest(t, "smarthubtest", bidder)
}
