package beachfront

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderAppnexus, config.Adapter{
		Endpoint:         `https://display.bfmio.com/prebid_display`,
		ExtraAdapterInfo: `{"video_endpoint":"https://reachms.bfmio.com/bid.json?exchange_id"}`,
	})
	adapterstest.RunJSONBidderTest(t, "beachfronttest", bidder)
}

func TestExtraInfoDefaultWhenEmpty(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAppnexus, config.Adapter{
		Endpoint:         `https://display.bfmio.com/prebid_display`,
		ExtraAdapterInfo: ``,
	})

	if buildErr != nil {
		t.Fatalf("Builder returned expected error %v", buildErr)
	}

	beachfrontBidder, castOK := bidder.(*BeachfrontAdapter)
	if !castOK {
		t.Fatal("Builder did not return a Beachfront Adapter")
	}

	assert.Equal(t, beachfrontBidder.extraInfo.VideoEndpoint, defaultVideoEndpoint)
}

func TestExtraInfoDefaultWhenNotSpecified(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAppnexus, config.Adapter{
		Endpoint:         `https://display.bfmio.com/prebid_display`,
		ExtraAdapterInfo: `{"video_endpoint":""}`,
	})

	if buildErr != nil {
		t.Fatalf("Builder returned expected error %v", buildErr)
	}

	beachfrontBidder, castOK := bidder.(*BeachfrontAdapter)
	if !castOK {
		t.Fatal("Builder did not return a Beachfront Adapter")
	}

	assert.Equal(t, beachfrontBidder.extraInfo.VideoEndpoint, defaultVideoEndpoint)
}

func TestExtraInfoMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAppnexus, config.Adapter{
		Endpoint:         `https://display.bfmio.com/prebid_display`,
		ExtraAdapterInfo: `malformed`,
	})

	assert.Error(t, buildErr)
}
