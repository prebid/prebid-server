package yaleo

import (
	"testing"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderYaleo, config.Adapter{
		Endpoint: "http://yaleo.example.com/prebid",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error: %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "yaleotest", bidder)
}

func TestBidderInfoOpenRTBVersion(t *testing.T) {
	infos, err := config.LoadBidderInfoFromDisk("../../static/bidder-info")
	if err != nil {
		t.Fatalf("failed to load bidder info: %v", err)
	}

	info, ok := infos["yaleo"]
	if !ok {
		t.Fatal("yaleo bidder info not found")
	}

	if info.OpenRTB == nil {
		t.Fatal("yaleo bidder info missing openrtb; ConvertDownTo25 would move user.eids to user.ext.eids")
	}

	if info.OpenRTB.Version != "2.6" {
		t.Fatalf("yaleo openrtb.version = %q, want 2.6 so user.eids stay at the path Yaleo reads", info.OpenRTB.Version)
	}
}
