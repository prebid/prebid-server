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
	bannerVideo := &openrtb2.Imp{Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}}
	bannerOnly := &openrtb2.Imp{Banner: &openrtb2.Banner{}}
	videoOnly := &openrtb2.Imp{Video: &openrtb2.Video{}}

	tests := []struct {
		name     string
		bid      openrtb2.Bid
		imp      *openrtb2.Imp
		expected openrtb_ext.BidType
		wantErr  bool
	}{
		{"mtype video wins over imp", openrtb2.Bid{MType: openrtb2.MarkupVideo, AdM: "<div>ad</div>"}, bannerVideo, openrtb_ext.BidTypeVideo, false},
		{"mtype banner wins over adm", openrtb2.Bid{MType: openrtb2.MarkupBanner, AdM: `<VAST version="4.0"></VAST>`}, bannerVideo, openrtb_ext.BidTypeBanner, false},
		{"no mtype, VAST adm on multiformat imp", openrtb2.Bid{AdM: `<VAST version="4.0"><Ad></Ad></VAST>`}, bannerVideo, openrtb_ext.BidTypeVideo, false},
		{"no mtype, lowercase vast adm", openrtb2.Bid{AdM: `<vast version="2.0"></vast>`}, bannerVideo, openrtb_ext.BidTypeVideo, false},
		{"no mtype, html adm on multiformat imp", openrtb2.Bid{AdM: "<div>ad</div>"}, bannerVideo, openrtb_ext.BidTypeBanner, false},
		{"no mtype, empty adm on multiformat imp", openrtb2.Bid{}, bannerVideo, openrtb_ext.BidTypeBanner, false},
		{"banner only imp ignores VAST adm", openrtb2.Bid{AdM: `<VAST version="4.0"></VAST>`}, bannerOnly, openrtb_ext.BidTypeBanner, false},
		{"video only imp ignores html adm", openrtb2.Bid{AdM: "<div>ad</div>"}, videoOnly, openrtb_ext.BidTypeVideo, false},
		{"VASTAdTagURI is not a VAST root", openrtb2.Bid{AdM: "<x><VASTAdTagURI>u</VASTAdTagURI></x>"}, bannerVideo, openrtb_ext.BidTypeBanner, false},
		{"mtype audio is dropped", openrtb2.Bid{MType: openrtb2.MarkupAudio, AdM: `<VAST version="4.0"></VAST>`}, bannerVideo, "", true},
		{"mtype native is dropped", openrtb2.Bid{MType: openrtb2.MarkupNative}, bannerVideo, "", true},
		{"unknown mtype is dropped", openrtb2.Bid{MType: 9}, videoOnly, "", true},
		{"negative mtype is dropped", openrtb2.Bid{MType: -1, AdM: "<div>ad</div>"}, bannerOnly, "", true},
		{"no mtype, VAST adm with leading whitespace and mixed case", openrtb2.Bid{AdM: "  \n<Vast version=\"4.2\"></Vast>"}, bannerVideo, openrtb_ext.BidTypeVideo, false},
		{"no mtype, self-closing VAST root", openrtb2.Bid{AdM: "<VAST/>"}, bannerVideo, openrtb_ext.BidTypeVideo, false},
		{"no mtype, space before VAST tag name", openrtb2.Bid{AdM: "< VAST version=\"3.0\"></VAST>"}, bannerVideo, openrtb_ext.BidTypeVideo, false},
		{"no mtype, VAST as text not as root tag", openrtb2.Bid{AdM: "<div>VAST</div>"}, bannerVideo, openrtb_ext.BidTypeBanner, false},
		{"no mtype, VASTX prefix is not a VAST root", openrtb2.Bid{AdM: "<VASTX></VASTX>"}, bannerVideo, openrtb_ext.BidTypeBanner, false},
		{"no mtype, empty adm on video only imp", openrtb2.Bid{}, videoOnly, openrtb_ext.BidTypeVideo, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mediaType, err := getMediaType(&tt.bid, []openrtb2.Imp{*tt.imp})
			assert.Equal(t, tt.expected, mediaType)
			if tt.wantErr {
				assert.IsType(t, &errortypes.BadServerResponse{}, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	mediaType, err := getMediaType(&openrtb2.Bid{ImpID: "missing"}, []openrtb2.Imp{*bannerVideo})
	assert.ErrorIs(t, err, errUnknownImp)
	assert.Equal(t, openrtb_ext.BidType(""), mediaType)

	// The same VAST markup resolves per impression when the request carries several.
	imps := []openrtb2.Imp{
		{ID: "banner-only", Banner: &openrtb2.Banner{}},
		{ID: "banner-video", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}},
	}
	vast := `<VAST version="4.0"></VAST>`
	mediaType, err = getMediaType(&openrtb2.Bid{ImpID: "banner-only", AdM: vast}, imps)
	assert.NoError(t, err)
	assert.Equal(t, openrtb_ext.BidTypeBanner, mediaType)
	mediaType, err = getMediaType(&openrtb2.Bid{ImpID: "banner-video", AdM: vast}, imps)
	assert.NoError(t, err)
	assert.Equal(t, openrtb_ext.BidTypeVideo, mediaType)
}
