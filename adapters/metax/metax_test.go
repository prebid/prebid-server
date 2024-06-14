package metax

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/util/ptrutil"
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
}

func TestAssignBannerSize(t *testing.T) {
	b1 := &openrtb2.Banner{
		Format: []openrtb2.Format{
			{W: 300, H: 250},
			{W: 728, H: 90},
		},
	}
	b1n, err := assignBannerSize(b1)
	assert.Equal(t, b1n.W, ptrutil.ToPtr(int64(300)))
	assert.Equal(t, b1n.H, ptrutil.ToPtr(int64(250)))
	assert.Nil(t, err)
	assert.NotSame(t, b1, b1n)

	b2 := &openrtb2.Banner{
		Format: []openrtb2.Format{
			{W: 300, H: 250},
			{W: 728, H: 90},
		},
		W: ptrutil.ToPtr(int64(1080)),
		H: ptrutil.ToPtr(int64(720)),
	}
	b2n, err := assignBannerSize(b2)
	assert.Equal(t, b2n.W, ptrutil.ToPtr(int64(1080)))
	assert.Equal(t, b2n.H, ptrutil.ToPtr(int64(720)))
	assert.Nil(t, err)
	assert.Same(t, b2, b2n)

	b3 := &openrtb2.Banner{
		W: ptrutil.ToPtr(int64(1080)),
		H: ptrutil.ToPtr(int64(720)),
	}
	b3n, err := assignBannerSize(b3)
	assert.Equal(t, b3n.W, ptrutil.ToPtr(int64(1080)))
	assert.Equal(t, b3n.H, ptrutil.ToPtr(int64(720)))
	assert.Nil(t, err)
	assert.Same(t, b3, b3n)
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

func TestBuilder(t *testing.T) {
	serverCfg := config.Server{}

	cfg1 := config.Adapter{Endpoint: "https://hb.metaxads.com/prebid"}
	_, err1 := Builder("test", cfg1, serverCfg)
	assert.Nil(t, err1)

	cfg2 := config.Adapter{Endpoint: ""}
	_, err2 := Builder("test2", cfg2, serverCfg)
	assert.NotNil(t, err2)
}
