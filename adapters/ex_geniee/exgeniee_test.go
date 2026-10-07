package exgeniee

import (
	"testing"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

const testEndpointTemplate = "https://aladdin.genieesspv.jp/yie/ld/exchange?id={{.PartnerId}}"

var testBidderName = openrtb_ext.BidderExGeniee

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(testBidderName, config.Adapter{Endpoint: testEndpointTemplate}, config.Server{})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "ex_genieetest", bidder)
}

func TestBuilderRejectsBrokenEndpointTemplate(t *testing.T) {
	_, err := Builder(testBidderName, config.Adapter{Endpoint: "https://example.com?id={{.PartnerId"}, config.Server{})
	assert.Error(t, err)
}
