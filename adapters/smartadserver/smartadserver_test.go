package smartadserver

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderSmartAdserver, config.Adapter{
		Endpoint: "https://ssb.smartadserver.com"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "smartadservertest", bidder)
}

func TestGetBidTypeFromMarkupType_WhenBanner_ShouldReturnBanner(t *testing.T) {
	mediaType := getBidTypeFromMarkupType(openrtb2.MarkupBanner)

	assert.Equal(t, openrtb_ext.BidTypeBanner, mediaType)
}

func TestGetBidTypeFromMarkupType_WhenVideo_ShouldReturnVideo(t *testing.T) {
	mediaType := getBidTypeFromMarkupType(openrtb2.MarkupVideo)

	assert.Equal(t, openrtb_ext.BidTypeVideo, mediaType)
}

func TestGetBidTypeFromMarkupType_WhenAudio_ShouldReturnAudio(t *testing.T) {
	mediaType := getBidTypeFromMarkupType(openrtb2.MarkupAudio)

	assert.Equal(t, openrtb_ext.BidTypeAudio, mediaType)
}

func TestGetBidTypeFromMarkupType_WhenNative_ShouldReturnNative(t *testing.T) {
	mediaType := getBidTypeFromMarkupType(openrtb2.MarkupNative)

	assert.Equal(t, openrtb_ext.BidTypeNative, mediaType)
}
