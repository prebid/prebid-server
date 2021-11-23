package beachfront

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBeachfront, config.Adapter{
		Endpoint:         `https://qa.beachrtb.com/prebid_display`,
		ExtraAdapterInfo: `{"video_endpoint":"https://qa.beachrtb.com/bid.json?exchange_id"}`,
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "beachfronttest", bidder)
}

func TestExtraInfoDefaultWhenEmpty(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBeachfront, config.Adapter{
		Endpoint:         `https://qa.beachrtb.com/prebid_display`,
		ExtraAdapterInfo: ``,
	})

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
	})

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
	})

	assert.Error(t, buildErr)
}
