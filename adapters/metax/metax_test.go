package metax

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderMetaX,
		config.Adapter{
			Endpoint: "https://hb.metaxads.com/prebid?sid={{.PublisherID}}&adunit={{.AdUnit}}&source=prebid-server",
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       1301,
			DataCenter:  "2",
		},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "metaxtest", bidder)
}

func TestParseBidderExt(t *testing.T) {
	data := `
{
  "bidder": {
    "publisherId": 1000,
    "adunit": 200
  }
}`
	imp := &openrtb2.Imp{
		Ext: json.RawMessage([]byte(data)),
	}
	metaxExt, err := parseBidderExt(imp)
	assert.Nil(t, err)
	assert.Equal(t, 1000, metaxExt.PublisherID)
	assert.Equal(t, 200, metaxExt.Adunit)
}

func TestPreprocessImp(t *testing.T) {
	assert.NotNil(t, preprocessImp(nil))

	imp1 := &openrtb2.Imp{
		Banner: &openrtb2.Banner{
			Format: []openrtb2.Format{
				{W: 300, H: 250},
				{W: 728, H: 90},
			},
		},
	}
	err1 := preprocessImp(imp1)
	assert.Nil(t, err1)

	imp2 := &openrtb2.Imp{
		Video: &openrtb2.Video{
			W: ptrutil.ToPtr(int64(1920)),
			H: ptrutil.ToPtr(int64(1920)),
		},
	}
	err2 := preprocessImp(imp2)
	assert.Nil(t, err2)
}

func TestAssignBannerSize(t *testing.T) {
	b1 := &openrtb2.Banner{
		Format: []openrtb2.Format{
			{W: 300, H: 250},
			{W: 728, H: 90},
		},
	}
	b1n := assignBannerSize(b1)
	assert.Equal(t, b1n.W, ptrutil.ToPtr(int64(300)))
	assert.Equal(t, b1n.H, ptrutil.ToPtr(int64(250)))
	assert.NotSame(t, b1, b1n)

	b2 := &openrtb2.Banner{
		Format: []openrtb2.Format{
			{W: 300, H: 250},
			{W: 728, H: 90},
		},
		W: ptrutil.ToPtr(int64(336)),
		H: ptrutil.ToPtr(int64(280)),
	}
	b2n := assignBannerSize(b2)
	assert.Equal(t, b2n.W, ptrutil.ToPtr(int64(336)))
	assert.Equal(t, b2n.H, ptrutil.ToPtr(int64(280)))
	assert.Same(t, b2, b2n)

	b3 := &openrtb2.Banner{
		W: ptrutil.ToPtr(int64(336)),
		H: ptrutil.ToPtr(int64(280)),
	}
	b3n := assignBannerSize(b3)
	assert.Equal(t, b3n.W, ptrutil.ToPtr(int64(336)))
	assert.Equal(t, b3n.H, ptrutil.ToPtr(int64(280)))
	assert.Same(t, b3, b3n)

	b4 := &openrtb2.Banner{
		Format: []openrtb2.Format{
			{W: 300, H: 250},
			{W: 728, H: 90},
		},
		W: ptrutil.ToPtr(int64(336)),
	}
	b4n := assignBannerSize(b4)
	assert.Equal(t, b4n.W, ptrutil.ToPtr(int64(300)))
	assert.Equal(t, b4n.H, ptrutil.ToPtr(int64(250)))
	assert.NotSame(t, b4, b4n)

	b5 := &openrtb2.Banner{}
	b5n := assignBannerSize(b5)
	assert.Nil(t, b5n.W)
	assert.Nil(t, b5n.H)
	assert.Same(t, b5, b5n)
}

func TestGetBidType(t *testing.T) {
	tests := []struct {
		bid     *openrtb2.Bid
		bidtype openrtb_ext.BidType
	}{
		{&openrtb2.Bid{AdM: "", MType: openrtb2.MarkupBanner}, openrtb_ext.BidTypeBanner},
		{&openrtb2.Bid{AdM: "", MType: openrtb2.MarkupVideo}, openrtb_ext.BidTypeVideo},
		{&openrtb2.Bid{AdM: "", MType: openrtb2.MarkupNative}, openrtb_ext.BidTypeNative},
		{&openrtb2.Bid{AdM: "", MType: openrtb2.MarkupAudio}, openrtb_ext.BidTypeAudio},
		{&openrtb2.Bid{AdM: "", MType: 0}, ""},
	}

	for _, test := range tests {
		bidType, err := getBidType(test.bid)
		assert.Equal(t, test.bidtype, bidType)
		if bidType == "" {
			assert.NotNil(t, err)
		}
	}
}

func TestGetBidVideo(t *testing.T) {
	tests := []struct {
		description string
		bid         *openrtb2.Bid
		bidvideo    openrtb_ext.ExtBidPrebidVideo
	}{
		{
			description: "One category, no duration",
			bid:         &openrtb2.Bid{Cat: []string{"IAB1-1"}},
			bidvideo:    openrtb_ext.ExtBidPrebidVideo{PrimaryCategory: "IAB1-1", Duration: 0},
		},
		{
			description: "Two categories and use the first, no duration",
			bid:         &openrtb2.Bid{Cat: []string{"IAB1-1", "IAB1-2"}},
			bidvideo:    openrtb_ext.ExtBidPrebidVideo{PrimaryCategory: "IAB1-1", Duration: 0},
		},
		{
			description: "No category, no duration",
			bid:         &openrtb2.Bid{Cat: []string{}},
			bidvideo:    openrtb_ext.ExtBidPrebidVideo{PrimaryCategory: "", Duration: 0},
		},
		{
			description: "No category(nil), no duration",
			bid:         &openrtb2.Bid{Cat: nil},
			bidvideo:    openrtb_ext.ExtBidPrebidVideo{PrimaryCategory: "", Duration: 0},
		},
		{
			description: "Two categories and use the first, duration is 15",
			bid:         &openrtb2.Bid{Cat: []string{"IAB1-1", "IAB1-2"}, Dur: 15},
			bidvideo:    openrtb_ext.ExtBidPrebidVideo{PrimaryCategory: "IAB1-1", Duration: 15},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			bidVideo := getBidVideo(test.bid)
			assert.Equal(t, test.bidvideo.PrimaryCategory, bidVideo.PrimaryCategory)
			assert.Equal(t, test.bidvideo.Duration, bidVideo.Duration)
		})
	}
}

func TestBuilder(t *testing.T) {
	serverCfg := config.Server{}

	cfg1 := config.Adapter{Endpoint: "https://hb.metaxads.com/prebid"}
	builder1, err1 := Builder("test", cfg1, serverCfg)
	assert.NotNil(t, builder1)
	assert.Nil(t, err1)

	// empty endpoint
	cfg2 := config.Adapter{Endpoint: ""}
	builder2, err2 := Builder("test2", cfg2, serverCfg)
	assert.Nil(t, builder2)
	assert.NotNil(t, err2)

	// invalid endpoint
	cfg3 := config.Adapter{Endpoint: "https://hb.metaxads.com/prebid?a={{}}"}
	builder3, err3 := Builder("test3", cfg3, serverCfg)
	assert.Nil(t, builder3)
	assert.NotNil(t, err3)
}

func TestMakeRequests(t *testing.T) {
	builder1, _ := Builder("metax", config.Adapter{Endpoint: "https://hb.metaxads.com/prebid?sid={{.PublisherId}}"}, config.Server{})
	reqDatas1, err1 := builder1.MakeRequests(&openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{
				Ext: []byte(`
					{
						"bidder": {
							"publisherId": 100,
							"adunit": 2
						}
					}
				`),
			},
		},
		Ext: []byte(`{invalid json}`),
	}, &adapters.ExtraRequestInfo{})
	assert.Equal(t, 0, len(reqDatas1))
	assert.Equal(t, 1, len(err1))

	builder2, _ := Builder(
		"metax",
		config.Adapter{Endpoint: "https://hb.metaxads.com/prebid?sid={{.PublisherID}}&adunit={{.AdUnit}}&source=prebid-server"},
		config.Server{},
	)
	reqDatas2, err2 := builder2.MakeRequests(&openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{
				Ext: []byte(`
					{
						"bidder": {
							"publisherId": 100,
							"adunit": 2
						}
					}
				`),
			},
		},
		Ext: []byte(`{invalid json}`),
	}, &adapters.ExtraRequestInfo{})
	assert.Equal(t, 0, len(reqDatas2))
	assert.Equal(t, 1, len(err2))
}
