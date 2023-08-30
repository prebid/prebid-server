package taboola

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

import (
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTaboola, config.Adapter{
		Endpoint: "http://{{.MediaType}}.whatever.com/{{.GvlID}}/{{.PublisherID}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 12, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "taboolatest", bidder)
}

func TestEmptyExternalUrl(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTaboola, config.Adapter{
		Endpoint: "http://whatever.com"}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderTaboola := bidder.(*adapter)

	assert.Equal(t, "", bidderTaboola.gvlID)
}
