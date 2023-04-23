package bidadjustments

import (
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestGetAdjustmentArray(t *testing.T) {
	testCases := []struct {
		name                   string
		givenRuleToAdjustments map[string][]openrtb_ext.Adjustments
		givenBidType           openrtb_ext.BidType
		givenBidderName        openrtb_ext.BidderName
		givenDealId            string
		expected               []openrtb_ext.Adjustments
	}{
		{
			name: "Priority #1 should be chosen",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustments{
				"banner|bidderA|dealId": {
					{
						AdjType: AdjTypeMultiplier,
						Value:   1.1,
					},
				},
				"banner|bidderA|*": {
					{
						AdjType: AdjTypeMultiplier,
						Value:   2.0,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustments{{AdjType: AdjTypeMultiplier, Value: 1.1}},
		},
		{
			name: "Priority #2 should be chosen",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustments{
				"banner|bidderA|*": {
					{
						AdjType: AdjTypeStatic,
						Value:   5.0,
					},
				},
				"banner|*|dealId": {
					{
						AdjType: AdjTypeMultiplier,
						Value:   2.0,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustments{{AdjType: AdjTypeStatic, Value: 5.0}},
		},
		{
			name: "Priority #3 should be chosen",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustments{
				"banner|*|dealId": {
					{
						AdjType:  AdjTypeCpm,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|bidderA|dealId": {
					{
						AdjType: AdjTypeMultiplier,
						Value:   1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustments{{AdjType: AdjTypeCpm, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority #4 should be chosen",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustments{
				"*|bidderA|dealId": {
					{
						AdjType:  AdjTypeCpm,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"banner|*|*": {
					{
						AdjType: AdjTypeMultiplier,
						Value:   1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustments{{AdjType: AdjTypeCpm, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority #5 should be chosen",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustments{
				"banner|*|*": {
					{
						AdjType:  AdjTypeCpm,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|bidderA|*": {
					{
						AdjType: AdjTypeMultiplier,
						Value:   1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustments{{AdjType: AdjTypeCpm, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority #6 should be chosen",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustments{
				"*|bidderA|*": {
					{
						AdjType:  AdjTypeCpm,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|*|dealId": {
					{
						AdjType: AdjTypeMultiplier,
						Value:   1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustments{{AdjType: AdjTypeCpm, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority #7 should be chosen",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustments{
				"*|*|dealId": {
					{
						AdjType:  AdjTypeCpm,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|*|*": {
					{
						AdjType: AdjTypeMultiplier,
						Value:   1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustments{{AdjType: AdjTypeCpm, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority #8 should be chosen, given the provided info doesn't match the other provided rules",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustments{
				"*|*|*": {
					{
						AdjType: AdjTypeMultiplier,
						Value:   1.1,
					},
				},
				"banner|bidderA|dealId": {
					{
						AdjType: AdjTypeMultiplier,
						Value:   1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderB",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustments{{AdjType: AdjTypeMultiplier, Value: 1.1}},
		},
		{
			name: "No dealID given, should choose correct rule",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustments{
				"banner|bidderA|*": {
					{
						AdjType: AdjTypeMultiplier,
						Value:   1.1,
					},
				},
				"banner|*|*": {
					{
						AdjType: AdjTypeMultiplier,
						Value:   1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "",
			expected:        []openrtb_ext.Adjustments{{AdjType: AdjTypeMultiplier, Value: 1.1}},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			adjArray := getAdjustmentArray(test.givenRuleToAdjustments, string(test.givenBidType), string(test.givenBidderName), test.givenDealId)
			assert.Equal(t, test.expected, adjArray, "Adjustment Array doesn't match")
		})
	}
}
