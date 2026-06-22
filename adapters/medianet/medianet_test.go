package medianet

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMedianet, config.Adapter{
		Endpoint:         "https://{{.Host}}/rtb/prebid",
		ExtraAdapterInfo: "http://localhost:8080/extrnal_url",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "medianettest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderMedianet, config.Adapter{
		Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestGetRegionHost(t *testing.T) {
	tests := []struct {
		region   string
		expected string
	}{
		{"USE", "prebid-adapter-useast.media.net"},
		{"USW", "prebid-adapter-uswest.media.net"},
		{"APAC", "prebid-adapter-asia.media.net"},
		{"EUC", "prebid-adapter-eu.media.net"},
		{"US", "prebid-adapter.media.net"},
		{"unknown", "prebid-adapter.media.net"},
		{"", "prebid-adapter.media.net"},
		{"use", "prebid-adapter-useast.media.net"},
	}
	for _, test := range tests {
		assert.Equal(t, test.expected, getRegionHost(test.region), "region: %q", test.region)
	}
}
