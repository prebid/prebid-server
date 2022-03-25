package exchange

import (
	"math"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestGetPriceBucketString(t *testing.T) {
	low := openrtb_ext.PriceGranularityFromString("low")
	medium := openrtb_ext.PriceGranularityFromString("medium")
	high := openrtb_ext.PriceGranularityFromString("high")
	auto := openrtb_ext.PriceGranularityFromString("auto")
	dense := openrtb_ext.PriceGranularityFromString("dense")
	custom1 := openrtb_ext.PriceGranularity{
		Precision: 2,
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

	// Define test cases
	type aTest struct {
		granularityId       string
		granularity         openrtb_ext.PriceGranularity
		expectedPriceBucket string
	}
	testGroups := []struct {
		groupDesc string
		cpm       float64
		testCases []aTest
	}{
		{
			groupDesc: "cpm below the max in every price bucket",
			cpm:       1.87,
			testCases: []aTest{
				{"low", low, "1.50"},
				{"medium", medium, "1.80"},
				{"high", high, "1.87"},
				{"auto", auto, "1.85"},
				{"dense", dense, "1.87"},
				{"custom1", custom1, "1.86"},
			},
		},
		{
			groupDesc: "cpm above the max in low price bucket",
			cpm:       5.72,
			testCases: []aTest{
				{"low", low, "5.00"},
				{"medium", medium, "5.70"},
				{"high", high, "5.72"},
				{"auto", auto, "5.70"},
				{"dense", dense, "5.70"},
				{"custom1", custom1, "5.70"},
			},
		},
		{
			groupDesc: "Precision value corner cases",
			cpm:       1.876,
			testCases: []aTest{
				{
					"Negative precision defaults to number of digits already in CPM float",
					openrtb_ext.PriceGranularity{Precision: -1, Ranges: []openrtb_ext.GranularityRange{{Max: 5, Increment: 0.05}}},
					"1.85",
				},
				{
					"Precision value equals zero, we expect to round up to the nearest integer",
					openrtb_ext.PriceGranularity{Ranges: []openrtb_ext.GranularityRange{{Max: 5, Increment: 0.05}}},
					"2",
				},
				{
					"Largest precision value PBS supports 15",
					openrtb_ext.PriceGranularity{Precision: 15, Ranges: []openrtb_ext.GranularityRange{{Max: 5, Increment: 0.05}}},
					"1.850000000000000",
				},
			},
		},
		{
			groupDesc: "Increment value corner cases",
			cpm:       1.876,
			testCases: []aTest{
				{
					"Negative increment, return empty string",
					openrtb_ext.PriceGranularity{Precision: 2, Ranges: []openrtb_ext.GranularityRange{{Max: 5, Increment: -0.05}}},
					"",
				},
				{
					"Zero increment, return empty string",
					openrtb_ext.PriceGranularity{Precision: 2, Ranges: []openrtb_ext.GranularityRange{{Max: 5}}},
					"",
				},
				{
					"Increment value is greater than CPM itself, return zero float value",
					openrtb_ext.PriceGranularity{Precision: 2, Ranges: []openrtb_ext.GranularityRange{{Max: 5, Increment: 1.877}}},
					"0.00",
				},
			},
		},
		{
			groupDesc: "Negative Cpm, return empty string since it does not belong into any range",
			cpm:       -1.876,
			testCases: []aTest{{"low", low, ""}},
		},
		{
			groupDesc: "Zero value Cpm, return the same, only in string format",
			cpm:       0,
			testCases: []aTest{{"low", low, "0.00"}},
		},
		{
			groupDesc: "Large Cpm, return bucket Max",
			cpm:       math.MaxFloat64,
			testCases: []aTest{{"low", low, "5.00"}},
		},
	}

	for _, testGroup := range testGroups {
		for _, test := range testGroup.testCases {
			priceBucket := GetPriceBucket(testGroup.cpm, test.granularity)

			assert.Equal(t, test.expectedPriceBucket, priceBucket, "Group: %s Granularity: %s :: Expected %s, got %s from %f", testGroup.groupDesc, test.granularityId, test.expectedPriceBucket, priceBucket, testGroup.cpm)
		}
	}
}
