package seedingAlliance

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderSeedingAlliance, config.Adapter{
		Endpoint: "https://mockup.seeding-alliance.de/",
	})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "seedingAlliancetest", bidder)
}

func TestResolvePriceMacro(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderSeedingAlliance, config.Adapter{
		Endpoint: "https://mockup.seeding-alliance.de/",
	})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adm := `{"link":{"url":"https://some_url.com/abc123?wp=${AUCTION_PRICE}"}`
	want := `{"link":{"url":"https://some_url.com/abc123?wp=12.34"}`

	bid := openrtb2.Bid{AdM: adm, Price: 12.34}
	resolvePriceMacro(&bid)

	if bid.AdM != want {
		t.Fatalf("want: %v, got: %v", want, bid.AdM)
	}
}

func TestGetMediaTypeForBid(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderSeedingAlliance, config.Adapter{
		Endpoint: "https://mockup.seeding-alliance.de/ssp-testing/native.html",
	})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	tests := []struct {
		name    string
		want    openrtb_ext.BidType
		wantErr bool
		bidType openrtb_ext.BidType
	}{
		{"native", openrtb_ext.BidTypeNative, false, openrtb_ext.BidTypeNative},
		{"banner", openrtb_ext.BidTypeBanner, false, openrtb_ext.BidTypeBanner},
		{"video", openrtb_ext.BidTypeVideo, false, openrtb_ext.BidTypeVideo},
		{"audio", openrtb_ext.BidTypeAudio, false, openrtb_ext.BidTypeAudio},
		{"empty type", "", true, ""},
	}

	for _, test := range tests {
		var bid openrtb2.SeatBid
		var extBid openrtb_ext.ExtBid

		if err := bid.Ext.UnmarshalJSON([]byte(`{"prebid": {"type":"` + string(test.bidType) + `"}}`)); err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		if err := json.Unmarshal(bid.Ext, &extBid); err != nil {
			t.Fatalf("could not unmarshal extBid: %v", err)
		}

		got, gotErr := getMediaTypeForBid(bid.Ext)
		assert.Equal(t, test.want, got)
		if gotErr != nil && !test.wantErr {
			t.Fatalf("wantErr: %v, gotErr: %v", test.wantErr, gotErr)
		}
	}
}

func TestAddTagID(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderSeedingAlliance, config.Adapter{
		Endpoint: "https://mockup.seeding-alliance.de/ssp-testing/native.html",
	})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	tests := []struct {
		name    string
		want    string
		data    string
		wantErr bool
	}{
		{"regular case", "abc123", "abc123", false},
		{"nil case", "", "", false},
		{"unmarshal err case", "", "", true},
	}

	for _, test := range tests {
		extSA, err := json.Marshal(openrtb_ext.ImpExtSeedingAlliance{AdUnitID: test.data})
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		extBidder, err := json.Marshal(adapters.ExtImpBidder{Bidder: extSA})
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		if test.wantErr {
			extBidder = []byte{}
		}

		ortbImp := openrtb2.Imp{Ext: extBidder}

		if err := addTagID(&ortbImp); err != nil {
			if test.wantErr {
				continue
			}
			t.Fatalf("unexpected error %v", err)
		}

		if test.want != ortbImp.TagID {
			t.Fatalf("want: %v, got: %v", test.want, ortbImp.TagID)
		}
	}
}

func TestCurExists(t *testing.T) {
	tests := []struct {
		name string
		cur  string
		data []string
		want bool
	}{
		{"no eur", "EUR", []string{"USD"}, false},
		{"eur exists", "EUR", []string{"USD", "EUR"}, true},
	}

	for _, test := range tests {
		got := curExists(test.data, test.cur)
		assert.Equal(t, test.want, got)
	}
}
