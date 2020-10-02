package exchange

import (
	"fmt"
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
		inGranularity       openrtb_ext.PriceGranularity
		expectedPriceBucket string
		expectedError       error
	}
	testGroups := []struct {
		groupDesc string
		inCpm     float64
		testCases []aTest
	}{
		{
			"cpm below the max in every price bucket",
			1.87,
			[]aTest{
				{"low", low, "1.50", nil},
				{"medium", medium, "1.80", nil},
				{"high", high, "1.87", nil},
				{"auto", auto, "1.85", nil},
				{"dense", dense, "1.87", nil},
				{"custom1", custom1, "1.86", nil},
			},
		},
		{
			"cpm above the max in low price bucket",
			5.72,
			[]aTest{
				{"low", low, "5.00", nil},
				{"medium", medium, "5.70", nil},
				{"high", high, "5.72", nil},
				{"auto", auto, "5.70", nil},
				{"dense", dense, "5.70", nil},
				{"custom1", custom1, "5.70", nil},
			},
		},
		{
			"Precision value corner cases",
			1.876,
			[]aTest{
				{
					"Negative precision defaults to number of digits already in CPM float",
					openrtb_ext.PriceGranularity{Precision: -1, Ranges: []openrtb_ext.GranularityRange{{Max: 5, Increment: 0.05}}},
					"1.85",
					nil,
				},
				{
					"Precision value equals zero, we expect to round up to the nearest integer",
					openrtb_ext.PriceGranularity{Ranges: []openrtb_ext.GranularityRange{{Max: 5, Increment: 0.05}}},
					"2",
					nil,
				},
				{
					"Very large precision value: expect error and an empty string in return",
					openrtb_ext.PriceGranularity{Precision: int(^uint(0) >> 1), Ranges: []openrtb_ext.GranularityRange{{Max: 5, Increment: 0.05}}},
					"",
					fmt.Errorf("Limit the number of precision figures to 4. Parsed value: %d", int(^uint(0)>>1)),
				},
			},
		},
		{
			"Increment value corner cases",
			1.876,
			[]aTest{
				{
					"Negative increment, return empty string",
					openrtb_ext.PriceGranularity{Precision: 2, Ranges: []openrtb_ext.GranularityRange{{Max: 5, Increment: -0.05}}},
					"",
					nil,
				},
				{
					"Zero increment, return empty string",
					openrtb_ext.PriceGranularity{Precision: 2, Ranges: []openrtb_ext.GranularityRange{{Max: 5}}},
					"",
					nil,
				},
				{
					"Increment value is greater than CPM itself, return zero float value",
					openrtb_ext.PriceGranularity{Precision: 2, Ranges: []openrtb_ext.GranularityRange{{Max: 5, Increment: 1.877}}},
					"0.00",
					nil,
				},
			},
		},
		{
			"Negative Cpm, return empty string since it does not belong into any range",
			-1.876,
			[]aTest{{"low", low, "", nil}},
		},
		{
			"Zero value Cpm, return the same, only in string format",
			0,
			[]aTest{{"low", low, "0.00", nil}},
		},
		{
			"Large Cpm, return bucket Max",
			math.MaxFloat64,
			[]aTest{{"low", low, "5.00", nil}},
		},
	}
	for _, testGroup := range testGroups {
		for _, test := range testGroup.testCases {
			priceBucket, err := GetCpmStringValue(testGroup.inCpm, test.inGranularity)
			assert.Equalf(t, test.expectedPriceBucket, priceBucket, "Group: %s Granularity: %s :: Expected %s, got %s from %f", testGroup.groupDesc, test.granularityId, test.expectedPriceBucket, priceBucket, testGroup.inCpm)
			assert.Equalf(t, test.expectedError, err, "Error value doesn't match")
		}
	}
}
