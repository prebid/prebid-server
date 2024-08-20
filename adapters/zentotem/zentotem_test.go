package zentotem

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderZentotem, config.Adapter{
		Endpoint: "https://rtb.zentotem.net/bid?sspuid=cqlnvfk00bhs0b6rci6g"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "zentotemtest", bidder)
}

func TestGetMediaTypeForBid(t *testing.T) {
	tests := []struct {
		name     string
		imp      openrtb2.Imp
		wantType openrtb_ext.BidType
		wantErr  bool
	}{
		{
			name: "get bid native type",
			imp: openrtb2.Imp{
				Native: &openrtb2.Native{
					Request: "test",
				},
			},
			wantType: openrtb_ext.BidTypeNative,
			wantErr:  false,
		},
		{
			name: "get bid banner type",
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{
					ID: "test",
				},
			},
			wantType: openrtb_ext.BidTypeBanner,
			wantErr:  false,
		},
		{
			name: "get bid video type",
			imp: openrtb2.Imp{
				Video: &openrtb2.Video{
					PodID: "test",
				},
			},
			wantType: openrtb_ext.BidTypeVideo,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bType, err := getMediaTypeForBid(tt.imp)
			if (err != nil) != tt.wantErr {
				t.Errorf("getMediaTypeForBid error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, bType, tt.wantType)
		})
	}
}
