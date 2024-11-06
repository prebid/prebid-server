package admatic

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdmatic, config.Adapter{
		Endpoint: "http://pbs.admatic.com.tr?host={{.Host}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1281, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "admatictest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAdmatic, config.Adapter{
		Endpoint: "host={{Host}}"}, config.Server{ExternalUrl: "http://hosturl.com"})

	assert.Error(t, buildErr)
}
