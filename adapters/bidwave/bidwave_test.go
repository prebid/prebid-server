package bidwave

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/currency"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBidwave, config.Adapter{
		Endpoint: "https://rtb.bidwave.net/rtb/v1/bid",
	}, config.Server{})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "bidwavetest", bidder)
}

func TestPrepareImpCurrencyConvertsToUSD(t *testing.T) {
	reqInfo := adapters.NewExtraRequestInfo(currency.NewRates(map[string]map[string]float64{
		"EUR": {
			"USD": 2,
		},
	}))
	imp := openrtb2.Imp{
		ID:          "imp-1",
		BidFloor:    1.5,
		BidFloorCur: "EUR",
	}

	preparedImp, err := prepareImpCurrency(imp, &reqInfo)

	assert.NoError(t, err)
	assert.Equal(t, 3.0, preparedImp.BidFloor)
	assert.Equal(t, "USD", preparedImp.BidFloorCur)
}

func TestPrepareImpCurrencyReturnsErrorWhenBidFloorCurrencyCannotBeConverted(t *testing.T) {
	reqInfo := adapters.NewExtraRequestInfo(currency.NewRates(map[string]map[string]float64{}))
	imp := openrtb2.Imp{
		ID:          "imp-1",
		BidFloor:    1.5,
		BidFloorCur: "EUR",
	}

	preparedImp, err := prepareImpCurrency(imp, &reqInfo)

	assert.Equal(t, imp, preparedImp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected currency USD for bid floor; unable to convert from EUR for impression imp-1")
}

func TestGetBidType(t *testing.T) {
	testCases := []struct {
		name        string
		bid         openrtb2.Bid
		imps        []openrtb2.Imp
		expected    openrtb_ext.BidType
		expectedErr string
	}{
		{
			name:     "banner mtype",
			bid:      openrtb2.Bid{ImpID: "imp-1", MType: openrtb2.MarkupBanner},
			expected: openrtb_ext.BidTypeBanner,
		},
		{
			name:     "video mtype",
			bid:      openrtb2.Bid{ImpID: "imp-1", MType: openrtb2.MarkupVideo},
			expected: openrtb_ext.BidTypeVideo,
		},
		{
			name: "missing mtype for banner-only impression",
			bid:  openrtb2.Bid{ImpID: "imp-1"},
			imps: []openrtb2.Imp{{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
			}},
			expected: openrtb_ext.BidTypeBanner,
		},
		{
			name: "missing mtype for video-only impression",
			bid:  openrtb2.Bid{ImpID: "imp-1"},
			imps: []openrtb2.Imp{{
				ID:    "imp-1",
				Video: &openrtb2.Video{},
			}},
			expected: openrtb_ext.BidTypeVideo,
		},
		{
			name: "missing mtype for multi-format impression",
			bid:  openrtb2.Bid{ImpID: "imp-1"},
			imps: []openrtb2.Imp{{
				ID:     "imp-1",
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
			}},
			expectedErr: "Bid must have non-zero MType for multi format impression with ID: \"imp-1\"",
		},
		{
			name:        "missing mtype for unknown impression",
			bid:         openrtb2.Bid{ImpID: "imp-1"},
			imps:        []openrtb2.Imp{{ID: "imp-2", Banner: &openrtb2.Banner{}}},
			expectedErr: "Failed to find impression for ID: \"imp-1\"",
		},
		{
			name:        "unsupported mtype",
			bid:         openrtb2.Bid{ImpID: "imp-1", MType: openrtb2.MarkupNative},
			expectedErr: "Unsupported MType 4 for impression with ID: \"imp-1\"",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			bidType, err := getBidType(&testCase.bid, testCase.imps)

			if testCase.expectedErr != "" {
				assert.EqualError(t, err, testCase.expectedErr)
				assert.Empty(t, bidType)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, bidType)
		})
	}
}
