package yahooAds

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestYahooAdsBidderEndpointConfig(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderYahooAds, config.Adapter{
		Endpoint: "http://localhost/bid",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderYahooAds := bidder.(*adapter)

	assert.Equal(t, "http://localhost/bid", bidderYahooAds.URI)
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderYahooAds, config.Adapter{}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "yahooAdstest", bidder)
}

func TestGetMediaType(t *testing.T) {
	bannerVideo := &openrtb2.Imp{ID: "imp", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}}
	bannerOnly := &openrtb2.Imp{ID: "imp", Banner: &openrtb2.Banner{}}
	videoOnly := &openrtb2.Imp{ID: "imp", Video: &openrtb2.Video{}}

	tests := []struct {
		name     string
		bid      openrtb2.Bid
		imp      *openrtb2.Imp
		expected openrtb_ext.BidType
		wantErr  string
	}{
		{"mtype video with VAST adm", openrtb2.Bid{MType: openrtb2.MarkupVideo, AdM: `<VAST version="4.0"><Ad></Ad></VAST>`}, bannerVideo, openrtb_ext.BidTypeVideo, ""},
		{"mtype banner with html adm", openrtb2.Bid{MType: openrtb2.MarkupBanner, AdM: "<div>ad</div>"}, bannerVideo, openrtb_ext.BidTypeBanner, ""},
		{"mtype video with nurl only, no adm", openrtb2.Bid{MType: openrtb2.MarkupVideo, NURL: "https://example.com/win"}, bannerVideo, openrtb_ext.BidTypeVideo, ""},
		{"no mtype, VAST adm on multiformat imp", openrtb2.Bid{AdM: `<VAST version="4.0"><Ad></Ad></VAST>`}, bannerVideo, openrtb_ext.BidTypeVideo, ""},
		{"no mtype, html adm on multiformat imp", openrtb2.Bid{AdM: "<div>ad</div>"}, bannerVideo, openrtb_ext.BidTypeBanner, ""},
		{"no mtype, empty adm on multiformat imp", openrtb2.Bid{}, bannerVideo, openrtb_ext.BidTypeBanner, ""},
		{"no mtype, html adm on banner-only imp", openrtb2.Bid{AdM: "<div>ad</div>"}, bannerOnly, openrtb_ext.BidTypeBanner, ""},
		{"no mtype, VAST adm on video-only imp", openrtb2.Bid{AdM: `<VAST version="4.0"></VAST>`}, videoOnly, openrtb_ext.BidTypeVideo, ""},
		{"VASTAdTagURI is not a VAST root", openrtb2.Bid{AdM: "<x><VASTAdTagURI>u</VASTAdTagURI></x>"}, bannerVideo, openrtb_ext.BidTypeBanner, ""},
		{"mtype audio is dropped", openrtb2.Bid{ID: "b", MType: openrtb2.MarkupAudio, AdM: `<VAST version="4.0"></VAST>`}, bannerVideo, "", "Unsupported mtype 3 for bid 'b'"},
		{"mtype native is dropped", openrtb2.Bid{ID: "b", MType: openrtb2.MarkupNative}, bannerVideo, "", "Unsupported mtype 4 for bid 'b'"},
		{"mtype native on banner-only imp is dropped", openrtb2.Bid{ID: "b", MType: openrtb2.MarkupNative}, bannerOnly, "", "Unsupported mtype 4 for bid 'b'"},
		{"mtype banner on banner-only imp", openrtb2.Bid{MType: openrtb2.MarkupBanner, AdM: "<div>ad</div>"}, bannerOnly, openrtb_ext.BidTypeBanner, ""},
		{"no mtype, VAST after xml declaration", openrtb2.Bid{AdM: `<?xml version="1.0" ?><VAST version="2.0"><Ad></Ad></VAST>`}, bannerVideo, openrtb_ext.BidTypeVideo, ""},
		{"no mtype, VAST attributes on next line", openrtb2.Bid{AdM: "<VAST\n  version=\"4.0\"><Ad></Ad></VAST>"}, bannerVideo, openrtb_ext.BidTypeVideo, ""},
		{"no mtype, lowercase vast is not a VAST root", openrtb2.Bid{AdM: `<vast version="4.0"></vast>`}, bannerVideo, openrtb_ext.BidTypeBanner, ""},
		{"no mtype, VAST as text not as root tag", openrtb2.Bid{AdM: "<div>VAST</div>"}, bannerVideo, openrtb_ext.BidTypeBanner, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.bid.ImpID = "imp"
			mediaType, err := getMediaType(&tt.bid, []openrtb2.Imp{*tt.imp})
			assert.Equal(t, tt.expected, mediaType)
			if tt.wantErr != "" {
				assert.IsType(t, &errortypes.BadServerResponse{}, err)
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	t.Run("unknown impression", func(t *testing.T) {
		mediaType, err := getMediaType(&openrtb2.Bid{ImpID: "missing"}, []openrtb2.Imp{*bannerVideo})
		assert.ErrorIs(t, err, errUnknownImp)
		assert.Equal(t, openrtb_ext.BidType(""), mediaType)
	})

	t.Run("uses the bid's impression", func(t *testing.T) {
		imps := []openrtb2.Imp{
			{ID: "banner-only", Banner: &openrtb2.Banner{}},
			{ID: "banner-video", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}},
		}
		vast := `<VAST version="4.0"></VAST>`
		mediaType, err := getMediaType(&openrtb2.Bid{ImpID: "banner-only", AdM: "<div>ad</div>"}, imps)
		assert.NoError(t, err)
		assert.Equal(t, openrtb_ext.BidTypeBanner, mediaType)
		mediaType, err = getMediaType(&openrtb2.Bid{ImpID: "banner-video", AdM: vast}, imps)
		assert.NoError(t, err)
		assert.Equal(t, openrtb_ext.BidTypeVideo, mediaType)
	})
}
