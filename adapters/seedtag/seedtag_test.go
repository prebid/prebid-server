package seedtag

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderSeedtag, config.Adapter{
		Endpoint: "https://s.seedtag.com"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "seedtagtest", bidder)
}

func TestGetMediaTypeForBid(t *testing.T) {
	tests := []struct {
		name           string
		want           openrtb_ext.BidType
		value          int8
		wantErr        bool
		wantErrContain string
	}{
		{
			name:           "invalie mediaType",
			want:           "",
			value:          0,
			wantErr:        true,
			wantErrContain: "bid.MType invalid",
		},
		{
			name:           "video mediaType",
			want:           openrtb_ext.BidTypeVideo,
			value:          2,
			wantErr:        false,
			wantErrContain: "",
		},
		{
			name:           "banner mediaType",
			want:           openrtb_ext.BidTypeBanner,
			value:          1,
			wantErr:        false,
			wantErrContain: "",
		},
	}

	for _, test := range tests {
		var bid openrtb2.Bid
		bid.MType = openrtb2.MarkupType(test.value)

		got, gotErr := getMediaTypeForBid(bid)
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
