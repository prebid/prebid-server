package yahooAds

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
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
	}{
		{"mtype video wins over imp", openrtb2.Bid{MType: openrtb2.MarkupVideo, AdM: "<div>ad</div>"}, bannerVideo, openrtb_ext.BidTypeVideo},
		{"mtype banner wins over adm", openrtb2.Bid{MType: openrtb2.MarkupBanner, AdM: `<VAST version="4.0"></VAST>`}, bannerVideo, openrtb_ext.BidTypeBanner},
		{"no mtype, VAST adm on multiformat imp", openrtb2.Bid{AdM: `<VAST version="4.0"><Ad></Ad></VAST>`}, bannerVideo, openrtb_ext.BidTypeVideo},
		{"no mtype, lowercase vast adm", openrtb2.Bid{AdM: `<vast version="2.0"></vast>`}, bannerVideo, openrtb_ext.BidTypeVideo},
		{"no mtype, html adm on multiformat imp", openrtb2.Bid{AdM: "<div>ad</div>"}, bannerVideo, openrtb_ext.BidTypeBanner},
		{"no mtype, empty adm on multiformat imp", openrtb2.Bid{}, bannerVideo, openrtb_ext.BidTypeBanner},
		{"banner only imp ignores VAST adm", openrtb2.Bid{AdM: `<VAST version="4.0"></VAST>`}, bannerOnly, openrtb_ext.BidTypeBanner},
		{"video only imp ignores html adm", openrtb2.Bid{AdM: "<div>ad</div>"}, videoOnly, openrtb_ext.BidTypeVideo},
		{"VASTAdTagURI is not a VAST root", openrtb2.Bid{AdM: "<x><VASTAdTagURI>u</VASTAdTagURI></x>"}, bannerVideo, openrtb_ext.BidTypeBanner},
		{"mtype audio is dropped", openrtb2.Bid{MType: openrtb2.MarkupAudio, AdM: `<VAST version="4.0"></VAST>`}, bannerVideo, ""},
		{"mtype native is dropped", openrtb2.Bid{MType: openrtb2.MarkupNative}, bannerVideo, ""},
		{"unknown mtype is dropped", openrtb2.Bid{MType: 9}, videoOnly, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, getMediaType(&tt.bid, tt.imp))
		})
	}
}
