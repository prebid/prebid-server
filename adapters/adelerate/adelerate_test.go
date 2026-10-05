package adelerate

import (
	"errors"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdelerate, config.Adapter{
		Endpoint: "https://pbs.bidelerate.com/openrtb2/auction"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "adeleratetest", bidder)
}

func TestGetMediaTypeForBid(t *testing.T) {
	tests := []struct {
		name             string
		bid              openrtb2.Bid
		expectedBidType  openrtb_ext.BidType
		expectedErrorMsg string
	}{
		{
			name:            "banner",
			bid:             openrtb2.Bid{ID: "banner-bid", MType: openrtb2.MarkupBanner},
			expectedBidType: openrtb_ext.BidTypeBanner,
		},
		{
			name:            "video",
			bid:             openrtb2.Bid{ID: "video-bid", MType: openrtb2.MarkupVideo},
			expectedBidType: openrtb_ext.BidTypeVideo,
		},
		{
			name:            "native",
			bid:             openrtb2.Bid{ID: "native-bid", MType: openrtb2.MarkupNative},
			expectedBidType: openrtb_ext.BidTypeNative,
		},
		{
			name:             "missing mtype",
			bid:              openrtb2.Bid{ID: "missing-mtype-bid"},
			expectedErrorMsg: "bid missing-mtype-bid missing required mtype",
		},
		{
			name:             "unsupported mtype",
			bid:              openrtb2.Bid{ID: "unsupported-mtype-bid", MType: 99},
			expectedErrorMsg: "unsupported mtype 99 for bid unsupported-mtype-bid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bidType, err := getMediaTypeForBid(test.bid)
			if test.expectedErrorMsg == "" {
				if err != nil {
					t.Fatalf("getMediaTypeForBid returned unexpected error: %v", err)
				}
				if bidType != test.expectedBidType {
					t.Errorf("getMediaTypeForBid returned bid type %q, expected %q", bidType, test.expectedBidType)
				}
				return
			}

			if err == nil {
				t.Fatal("getMediaTypeForBid expected an error, got nil")
			}
			if err.Error() != test.expectedErrorMsg {
				t.Errorf("getMediaTypeForBid returned error %q, expected %q", err.Error(), test.expectedErrorMsg)
			}

			var badServerResponse *errortypes.BadServerResponse
			if !errors.As(err, &badServerResponse) {
				t.Errorf("getMediaTypeForBid returned error type %T, expected *errortypes.BadServerResponse", err)
			}
		})
	}
}
