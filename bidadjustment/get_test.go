package bidadjustment

import (
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestGetAdjustmentArray(t *testing.T) {
	testCases := []struct {
		name                   string
		givenRuleToAdjustments map[string][]openrtb_ext.Adjustment
		givenBidType           openrtb_ext.BidType
		givenBidderName        openrtb_ext.BidderName
		givenDealId            string
		expected               []openrtb_ext.Adjustment
	}{
		{
			name: "Priority1",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|bidderA|dealId": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
				"banner|bidderA|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 2.0,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}},
		},
		{
			name: "Priority2",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|bidderA|*": {
					{
						Type:  AdjustmentTypeStatic,
						Value: 5.0,
					},
				},
				"banner|*|dealId": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 2.0,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeStatic, Value: 5.0}},
		},
		{
			name: "Priority3",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|*|dealId": {
					{
						Type:     AdjustmentTypeCpm,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|bidderA|dealId": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCpm, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority4",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|bidderA|dealId": {
					{
						Type:     AdjustmentTypeCpm,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"banner|*|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCpm, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority5",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|*|*": {
					{
						Type:     AdjustmentTypeCpm,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|bidderA|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCpm, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority6",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|bidderA|*": {
					{
						Type:     AdjustmentTypeCpm,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|*|dealId": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCpm, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority7",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|*|dealId": {
					{
						Type:     AdjustmentTypeCpm,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|*|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCpm, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority8",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|*|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
				"banner|bidderA|dealId": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderB",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}},
		},
		{
			name: "NoDealID",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|bidderA|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
				"banner|*|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			adjArray := get(test.givenRuleToAdjustments, string(test.givenBidType), string(test.givenBidderName), test.givenDealId)
			assert.Equal(t, test.expected, adjArray, "Adjustment Array doesn't match")
		})
	}
}
