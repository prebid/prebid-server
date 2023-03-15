package limelightDigital

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderLimelightDigital, config.Adapter{
		Endpoint: "http://{{.Host}}.ortb.net/openrtb/{{.PublisherID}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "LimelightDigitaltest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderLimelightDigital, config.Adapter{
		Endpoint: "{{Malformed}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestBadConfig(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderLimelightDigital, config.Adapter{
		ExtraAdapterInfo: `{foo:42}`,
	},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}
