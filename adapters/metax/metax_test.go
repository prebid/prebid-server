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
    "publisherId": "1000",
    "adunit": "200"
  }
}`
	imp := &openrtb2.Imp{
		Ext: json.RawMessage([]byte(data)),
	}
	metaxExt, err := parseBidderExt(imp)
	assert.Nil(t, err)
	assert.Equal(t, "1000", metaxExt.PublisherID)
	assert.Equal(t, "200", metaxExt.Adunit)
}

func TestValidateParams(t *testing.T) {
	tests := []struct {
		ext      *openrtb_ext.ExtImpMetaX
		errorMsg string
	}{
		{
			ext: &openrtb_ext.ExtImpMetaX{
				PublisherID: "1000",
				Adunit:      "1",
			},
			errorMsg: "",
		},
		{
			ext: &openrtb_ext.ExtImpMetaX{
				PublisherID: "1000",
				Adunit:      "abc",
			},
			errorMsg: "invalid adunit",
		},
		{
			ext: &openrtb_ext.ExtImpMetaX{
				PublisherID: "abc",
				Adunit:      "1",
			},
			errorMsg: "invalid publisher ID",
		},
	}

	for _, test := range tests {
		errMsg := ""
		err := validateParams(test.ext)
		if err != nil {
			errMsg = err.Error()
		}
		assert.Equal(t, test.errorMsg, errMsg)
	}
}

func TestPreprocessImp(t *testing.T) {
	assert.NotNil(t, preprocessImp(nil))

	imp := &openrtb2.Imp{
		Banner: &openrtb2.Banner{},
		Video:  &openrtb2.Video{},
	}
	preprocessImp(imp)
	assert.Nil(t, imp.Banner)
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
}

func TestGetBidType(t *testing.T) {
	imps := []openrtb2.Imp{
		{ID: "1", Banner: &openrtb2.Banner{}},
		{ID: "2", Video: &openrtb2.Video{}},
		{ID: "3", Native: &openrtb2.Native{}},
		{ID: "4", Audio: &openrtb2.Audio{}},
	}

	tests := []struct {
		id      string
		bidtype openrtb_ext.BidType
	}{
		{"1", openrtb_ext.BidTypeBanner},
		{"2", openrtb_ext.BidTypeVideo},
		{"3", openrtb_ext.BidTypeNative},
		{"4", openrtb_ext.BidTypeAudio},
		{"unknown", openrtb_ext.BidTypeBanner},
	}

	for _, test := range tests {
		assert.Equal(t, test.bidtype, getBidType(imps, test.id))
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
