package seedingAlliance

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderSeedingAlliance, config.Adapter{
		Endpoint: "https://mockup.seeding-alliance.de/",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "seedingAlliancetest", bidder)
}

func TestResolvePriceMacro(t *testing.T) {
	adm := `{"link":{"url":"https://some_url.com/abc123?wp=${AUCTION_PRICE}"}`
	want := `{"link":{"url":"https://some_url.com/abc123?wp=12.34"}`

	bid := openrtb2.Bid{AdM: adm, Price: 12.34}
	resolvePriceMacro(&bid)

	if bid.AdM != want {
		t.Fatalf("want: %v, got: %v", want, bid.AdM)
	}
}

func TestGetMediaTypeForBid(t *testing.T) {
	tests := []struct {
		name           string
		want           openrtb_ext.BidType
		invalidJSON    bool
		wantErr        bool
		wantErrContain string
		bidType        openrtb_ext.BidType
	}{
		{
			name:           "native",
			want:           openrtb_ext.BidTypeNative,
			invalidJSON:    false,
			wantErr:        false,
			wantErrContain: "",
			bidType:        "native",
		},
		{
			name:           "banner",
			want:           openrtb_ext.BidTypeBanner,
			invalidJSON:    false,
			wantErr:        false,
			wantErrContain: "",
			bidType:        "banner",
		},
		{
			name:           "video",
			want:           openrtb_ext.BidTypeVideo,
			invalidJSON:    false,
			wantErr:        false,
			wantErrContain: "",
			bidType:        "video",
		},
		{
			name:           "audio",
			want:           openrtb_ext.BidTypeAudio,
			invalidJSON:    false,
			wantErr:        false,
			wantErrContain: "",
			bidType:        "audio",
		},
		{
			name:           "empty type",
			want:           "",
			invalidJSON:    false,
			wantErr:        true,
			wantErrContain: "invalid BidType",
			bidType:        "",
		},
		{
			name:           "invalid type",
			want:           "",
			invalidJSON:    false,
			wantErr:        true,
			wantErrContain: "invalid BidType",
			bidType:        "invalid",
		},
		{
			name:           "invalid json",
			want:           "",
			invalidJSON:    true,
			wantErr:        true,
			wantErrContain: "bid.Ext.Prebid is empty",
			bidType:        "",
		},
	}

	for _, test := range tests {
		var bid openrtb2.SeatBid
		var extBid openrtb_ext.ExtBid

		var bidExtJsonString string
		if test.invalidJSON {
			bidExtJsonString = `{"x_prebid": {"type":""}}`
		} else {
			bidExtJsonString = `{"prebid": {"type":"` + string(test.bidType) + `"}}`
		}

		if err := bid.Ext.UnmarshalJSON([]byte(bidExtJsonString)); err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		if err := json.Unmarshal(bid.Ext, &extBid); err != nil {
			t.Fatalf("could not unmarshal extBid: %v", err)
		}

		got, gotErr := getMediaTypeForBid(bid.Ext)
		assert.Equal(t, test.want, got)

		if test.wantErr {
			if gotErr != nil {
				assert.Contains(t, gotErr.Error(), test.wantErrContain)
				continue
			}
			t.Fatalf("wantErr: %v, gotErr: %v", test.wantErr, gotErr)
		}
	}
}

func TestAddTagID(t *testing.T) {
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
