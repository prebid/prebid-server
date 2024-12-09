package exchange

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestGetPriceBucketString(t *testing.T) {
	low, _ := openrtb_ext.NewPriceGranularityFromLegacyID("low")
	medium, _ := openrtb_ext.NewPriceGranularityFromLegacyID("medium")
	high, _ := openrtb_ext.NewPriceGranularityFromLegacyID("high")
	auto, _ := openrtb_ext.NewPriceGranularityFromLegacyID("auto")
	dense, _ := openrtb_ext.NewPriceGranularityFromLegacyID("dense")

	custom1 := openrtb_ext.PriceGranularity{
		Precision: ptrutil.ToPtr(2),
		Ranges: []openrtb_ext.GranularityRange{
			{
				Min:       0.0,
				Max:       5.0,
				Increment: 0.03,
			},
			{
				Min:       5.0,
				Max:       10.0,
				Increment: 0.1,
			},
		},
	}

	custom2 := openrtb_ext.PriceGranularity{
		Precision: ptrutil.ToPtr(2),
		Ranges: []openrtb_ext.GranularityRange{
			{
				Min:       0.0,
				Max:       1.5,
				Increment: 1.0,
			},
			{
				Min:       1.5,
				Max:       10.0,
				Increment: 1.2,
			},
		},
	}

	// Define test cases
	type aTest struct {
		granularityId       string
		targetData          targetData
		expectedPriceBucket string
	}
	testGroups := []struct {
		groupDesc string
		bid       openrtb2.Bid
		testCases []aTest
	}{
		{
			groupDesc: "cpm below the max in every price bucket",
			bid:       openrtb2.Bid{Price: 1.87},
			testCases: []aTest{
				{"low", targetData{priceGranularity: low}, "1.50"},
				{"medium", targetData{priceGranularity: medium}, "1.80"},
				{"high", targetData{priceGranularity: high}, "1.87"},
				{"auto", targetData{priceGranularity: auto}, "1.85"},
				{"dense", targetData{priceGranularity: dense}, "1.87"},
				{"custom1", targetData{priceGranularity: custom1}, "1.86"},
				{"custom2", targetData{priceGranularity: custom2}, "1.50"},
			},
		},
		{
			groupDesc: "cpm above the max in low price bucket",
			bid:       openrtb2.Bid{Price: 5.72},
			testCases: []aTest{
				{"low", targetData{priceGranularity: low}, "5.00"},
				{"medium", targetData{priceGranularity: medium}, "5.70"},
				{"high", targetData{priceGranularity: high}, "5.72"},
				{"auto", targetData{priceGranularity: auto}, "5.70"},
				{"dense", targetData{priceGranularity: dense}, "5.70"},
				{"custom1", targetData{priceGranularity: custom1}, "5.70"},
				{"custom2", targetData{priceGranularity: custom2}, "5.10"},
			},
		},
		{
			groupDesc: "media type price granularity for bid type video",
			bid:       openrtb2.Bid{Price: 5.0, MType: openrtb2.MarkupVideo},
			testCases: []aTest{
				{"medium", targetData{priceGranularity: medium}, "5.00"},
				{"video-custom2", targetData{priceGranularity: medium, mediaTypePriceGranularity: openrtb_ext.MediaTypePriceGranularity{Video: &custom2}}, "3.90"},
				{"banner-custom2", targetData{priceGranularity: medium, mediaTypePriceGranularity: openrtb_ext.MediaTypePriceGranularity{Banner: &custom2}}, "5.00"},
			},
		},
		{
			groupDesc: "media type price granularity for bid type banner",
			bid:       openrtb2.Bid{Price: 5.0, MType: openrtb2.MarkupBanner},
			testCases: []aTest{
				{"medium", targetData{priceGranularity: medium}, "5.00"},
				{"video-custom2", targetData{priceGranularity: medium, mediaTypePriceGranularity: openrtb_ext.MediaTypePriceGranularity{Video: &custom2}}, "5.00"},
				{"banner-custom2", targetData{priceGranularity: medium, mediaTypePriceGranularity: openrtb_ext.MediaTypePriceGranularity{Banner: &custom2}}, "3.90"},
			},
		},
		{
			groupDesc: "media type price granularity for bid type native",
			bid:       openrtb2.Bid{Price: 5.0, MType: openrtb2.MarkupNative},
			testCases: []aTest{
				{"medium", targetData{priceGranularity: medium}, "5.00"},
				{"video-custom2", targetData{priceGranularity: medium, mediaTypePriceGranularity: openrtb_ext.MediaTypePriceGranularity{Video: &custom2}}, "5.00"},
				{"native-custom2", targetData{priceGranularity: medium, mediaTypePriceGranularity: openrtb_ext.MediaTypePriceGranularity{Native: &custom2}}, "3.90"},
			},
		},
		{
			groupDesc: "media type price granularity set but bid type incorrect",
			bid:       openrtb2.Bid{Price: 5.0, Ext: json.RawMessage(`{`)},
			testCases: []aTest{
				{"medium", targetData{priceGranularity: medium}, "5.00"},
				{"video-custom2", targetData{priceGranularity: medium, mediaTypePriceGranularity: openrtb_ext.MediaTypePriceGranularity{Video: &custom2}}, "5.00"},
				{"banner-custom2", targetData{priceGranularity: medium, mediaTypePriceGranularity: openrtb_ext.MediaTypePriceGranularity{Banner: &custom2}}, "5.00"},
				{"native-custom2", targetData{priceGranularity: medium, mediaTypePriceGranularity: openrtb_ext.MediaTypePriceGranularity{Native: &custom2}}, "5.00"},
			},
		},
		{
			groupDesc: "cpm equal the max for custom granularity",
			bid:       openrtb2.Bid{Price: 10},
			testCases: []aTest{
				{"custom1", targetData{priceGranularity: custom1}, "10.00"},
				{"custom2", targetData{priceGranularity: custom2}, "9.90"},
			},
		},
		{
			groupDesc: "Precision value corner cases",
			bid:       openrtb2.Bid{Price: 1.876},
			testCases: []aTest{
				{
					"Negative precision defaults to number of digits already in CPM float",
					targetData{priceGranularity: openrtb_ext.PriceGranularity{Precision: ptrutil.ToPtr(-1), Ranges: []openrtb_ext.GranularityRange{{Max: 5, Increment: 0.05}}}},
					"1.85",
				},
				{
					"Precision value equals zero, we expect to round up to the nearest integer",
					targetData{priceGranularity: openrtb_ext.PriceGranularity{Precision: ptrutil.ToPtr(0), Ranges: []openrtb_ext.GranularityRange{{Max: 5, Increment: 0.05}}}},
					"2",
				},
				{
					"Largest precision value PBS supports 15",
					targetData{priceGranularity: openrtb_ext.PriceGranularity{Precision: ptrutil.ToPtr(15), Ranges: []openrtb_ext.GranularityRange{{Max: 5, Increment: 0.05}}}},
					"1.850000000000000",
				},
			},
		},
		{
			groupDesc: "Increment value corner cases",
			bid:       openrtb2.Bid{Price: 1.876},
			testCases: []aTest{
				{
					"Negative increment, return empty string",
					targetData{priceGranularity: openrtb_ext.PriceGranularity{Precision: ptrutil.ToPtr(2), Ranges: []openrtb_ext.GranularityRange{{Max: 5, Increment: -0.05}}}},
					"",
				},
				{
					"Zero increment, return empty string",
					targetData{priceGranularity: openrtb_ext.PriceGranularity{Precision: ptrutil.ToPtr(2), Ranges: []openrtb_ext.GranularityRange{{Max: 5}}}},
					"",
				},
				{
					"Increment value is greater than CPM itself, return zero float value",
					targetData{priceGranularity: openrtb_ext.PriceGranularity{Precision: ptrutil.ToPtr(2), Ranges: []openrtb_ext.GranularityRange{{Max: 5, Increment: 1.877}}}},
					"0.00",
				},
			},
		},
		{
			groupDesc: "Negative Cpm, return empty string since it does not belong into any range",
			bid:       openrtb2.Bid{Price: -1.876},
			testCases: []aTest{{"low", targetData{priceGranularity: low}, ""}},
		},
		{
			groupDesc: "Zero value Cpm, return the same, only in string format",
			bid:       openrtb2.Bid{Price: 0},
			testCases: []aTest{{"low", targetData{priceGranularity: low}, "0.00"}},
		},
		{
			groupDesc: "Large Cpm, return bucket Max",
			bid:       openrtb2.Bid{Price: math.MaxFloat64},
			testCases: []aTest{{"low", targetData{priceGranularity: low}, "5.00"}},
		},
	}

	for _, testGroup := range testGroups {
		for i, test := range testGroup.testCases {
			var priceBucket string
			assert.NotPanics(t, func() { priceBucket = GetPriceBucket(testGroup.bid, test.targetData) }, "Group: %s Granularity: %d", testGroup.groupDesc, i)
			assert.Equal(t, test.expectedPriceBucket, priceBucket, "Group: %s Granularity: %s :: Expected %s, got %s from %f", testGroup.groupDesc, test.granularityId, test.expectedPriceBucket, priceBucket, testGroup.bid.Price)
		}
	}
}
