package aniview

import (
	"testing"

	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestBuildRequestExtInvalid(t *testing.T) {
	impExt := &openrtb_ext.ImpExtAniview{PublisherId: "pub", ChannelId: "chan"}
	if _, err := buildRequestExt([]byte(`"not-an-object"`), impExt); err == nil {
		t.Error("expected error for non-object request.ext")
	}
	ext, err := buildRequestExt([]byte(`{"prebid":{"debug":true}}`), impExt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(ext) == "" {
		t.Error("expected merged ext")
	}
}

func TestMakeBidsEmptyBody(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderAniview, config.Adapter{Endpoint: "https://rtb.aniview.com/sspRTB2"}, config.Server{})
	for _, body := range []string{"", "\n", "  \n"} {
		resp, errs := bidder.(*adapter).MakeBids(nil, &adapters.RequestData{Body: []byte("{}")}, &adapters.ResponseData{StatusCode: 200, Body: []byte(body)})
		if resp != nil || errs != nil {
			t.Errorf("empty body %q should be a silent no-bid, got resp=%v errs=%v", body, resp, errs)
		}
	}
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAniview, config.Adapter{
		Endpoint: "https://rtb.aniview.com/sspRTB2",
	},
		config.Server{
			ExternalUrl: "http://hosturl.com", GvlID: 780, DataCenter: "2",
		})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "aniviewtest", bidder)
}
