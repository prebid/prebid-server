package insticator

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderInsticator, config.Adapter{
		Endpoint:         "https://insticator.example.com/v1/pbs",
		ExtraAdapterInfo: `{"app_endpoint": "https://insticator-app.example.com/v1/pbs"}`,
	},
		config.Server{ExternalUrl: "https://insticator.example.com/v1/pbs", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "insticatortest", bidder)
}

func TestGetMediaTypeForBid(t *testing.T) {
	tests := []struct {
		name         string
		mType        openrtb2.MarkupType
		bidExt       json.RawMessage
		expectedType openrtb_ext.BidType
	}{
		{
			name:         "banner markup maps to banner",
			mType:        openrtb2.MarkupBanner,
			expectedType: openrtb_ext.BidTypeBanner,
		},
		{
			name:         "video markup maps to video",
			mType:        openrtb2.MarkupVideo,
			expectedType: openrtb_ext.BidTypeVideo,
		},
		{
			name:         "audio markup maps to audio",
			mType:        openrtb2.MarkupAudio,
			expectedType: openrtb_ext.BidTypeAudio,
		},
		{
			name:         "markup wins over a conflicting ext media type",
			mType:        openrtb2.MarkupAudio,
			bidExt:       json.RawMessage(`{"insticator":{"mediaType":"banner"}}`),
			expectedType: openrtb_ext.BidTypeAudio,
		},
		{
			name:         "absent markup falls back to banner",
			mType:        0,
			expectedType: openrtb_ext.BidTypeBanner,
		},
		{
			name:         "absent markup falls back to banner even when the ext names a media type",
			mType:        0,
			bidExt:       json.RawMessage(`{"insticator":{"mediaType":"audio"}}`),
			expectedType: openrtb_ext.BidTypeBanner,
		},
		{
			name:         "native markup maps to native",
			mType:        openrtb2.MarkupNative,
			expectedType: openrtb_ext.BidTypeNative,
		},
		{
			name:         "a markup type the adapter does not serve falls back to banner",
			mType:        openrtb2.MarkupType(99),
			expectedType: openrtb_ext.BidTypeBanner,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bid := &openrtb2.Bid{ImpID: "imp-1", MType: test.mType, Ext: test.bidExt}
			assert.Equal(t, test.expectedType, getMediaTypeForBid(bid))
		})
	}
}

func TestGetBidMetaMediaType(t *testing.T) {
	tests := []struct {
		name         string
		bidType      openrtb_ext.BidType
		expectedType string
	}{
		{name: "audio", bidType: openrtb_ext.BidTypeAudio, expectedType: "audio"},
		{name: "video", bidType: openrtb_ext.BidTypeVideo, expectedType: "video"},
		{name: "banner", bidType: openrtb_ext.BidTypeBanner, expectedType: "banner"},
		{name: "native", bidType: openrtb_ext.BidTypeNative, expectedType: "native"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bid := &openrtb2.Bid{ADomain: []string{"insticator.com"}, Cat: []string{"IAB1-1"}}
			meta := getBidMeta(bid, test.bidType, "insticator")

			assert.Equal(t, test.expectedType, meta.MediaType)
			assert.Equal(t, "insticator", meta.Seat)
			assert.Equal(t, []string{"insticator.com"}, meta.AdvertiserDomains)
			assert.Equal(t, "IAB1-1", meta.PrimaryCategoryID)
		})
	}
}

func TestGetBidVideoOnlyForVideo(t *testing.T) {
	bid := &openrtb2.Bid{Dur: 30, Cat: []string{"IAB1-1"}}

	assert.Nil(t, getBidVideo(bid, openrtb_ext.BidTypeAudio), "audio bids carry no bid video")
	assert.Nil(t, getBidVideo(bid, openrtb_ext.BidTypeBanner), "banner bids carry no bid video")

	video := getBidVideo(bid, openrtb_ext.BidTypeVideo)
	assert.NotNil(t, video)
	assert.Equal(t, 30, video.Duration)
	assert.Equal(t, "IAB1-1", video.PrimaryCategory)
}
