package zentotem

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderZentotem, config.Adapter{
		Endpoint: "http://localhost/bid?sspuid=prebid"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "zentotemtest", bidder)
}

func TestGetMediaTypeForBid(t *testing.T) {
	tests := []struct {
		name     string
		bid      openrtb2.Bid
		wantType openrtb_ext.BidType
		wantErr  bool
	}{
		{
			name: "get_bid_native_type",
			bid: openrtb2.Bid{
				MType: 4,
			},
			wantType: openrtb_ext.BidTypeNative,
			wantErr:  false,
		},
		{
			name: "get_bid_banner_type",
			bid: openrtb2.Bid{
				MType: 1,
			},
			wantType: openrtb_ext.BidTypeBanner,
			wantErr:  false,
		},
		{
			name: "get_bid_video_type",
			bid: openrtb2.Bid{
				MType: 2,
			},
			wantType: openrtb_ext.BidTypeVideo,
			wantErr:  false,
		},
		{
			name:     "fail",
			wantType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bType, err := getMediaTypeForBid(tt.bid)
			if (err != nil) != tt.wantErr {
				t.Errorf("getMediaTypeForBid error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, bType, tt.wantType)
		})
	}
}
