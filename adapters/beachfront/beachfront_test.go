package beachfront

import (
	"testing"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBeachfront, config.Adapter{
		Endpoint:         `https://qa.beachrtb.com/prebid_display`,
		ExtraAdapterInfo: `{"video_endpoint":"https://qa.beachrtb.com/bid.json?exchange_id"}`,
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "beachfronttest", bidder)
}

func TestExtraInfoDefaultWhenEmpty(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBeachfront, config.Adapter{
		Endpoint:         `https://qa.beachrtb.com/prebid_display`,
		ExtraAdapterInfo: ``,
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderBeachfront, _ := bidder.(*BeachfrontAdapter)

	assert.Equal(t, bidderBeachfront.extraInfo.VideoEndpoint, defaultVideoEndpoint)
}

func TestExtraInfoDefaultWhenNotSpecified(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBeachfront, config.Adapter{
		Endpoint:         `https://qa.beachrtb.com/prebid_display`,
		ExtraAdapterInfo: `{"video_endpoint":""}`,
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderBeachfront, _ := bidder.(*BeachfrontAdapter)

	assert.Equal(t, bidderBeachfront.extraInfo.VideoEndpoint, defaultVideoEndpoint)
}

func TestExtraInfoMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderBeachfront, config.Adapter{
		Endpoint:         `https://qa.beachrtb.com/prebid_display`,
		ExtraAdapterInfo: `malformed`,
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestExtractNurlVideoCrid(t *testing.T) {
	tests := []struct {
		name     string
		nurl     string
		expected string
	}{
		{
			name:     "valid nurl",
			nurl:     "https://useast.bfmio.com/getBids?aid=bid:70b99087-1b92-4e81-bc42-05c940fd6014:bid-id",
			expected: "70b99087-1b92-4e81-bc42-05c940fd6014",
		},
		{
			name: "nurl with no colons",
			nurl: "short-nurl",
		},
		{
			name: "nurl with one colon",
			nurl: "short:nurl",
		},
		{
			name: "empty nurl",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, extractNurlVideoCrid(test.nurl))
		})
	}
}
