package adverxo

import (
	"testing"

	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdverxo, config.Adapter{
		Endpoint: "https://example.com/auction?id={{.AdUnit}}&auth={{.TokenID}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "adverxotest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAdverxo, config.Adapter{
		Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}
