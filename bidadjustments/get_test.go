package bidadjustments

import (
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestGetAdjustmentArray(t *testing.T) {
	testCases := []struct {
		name                string
		givenBidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments
		givenBidType        openrtb_ext.BidType
		givenBidderName     openrtb_ext.BidderName
		givenDealId         string
		expected            []openrtb_ext.Adjustments
	}{
		{
			name: "One bid adjustment, should return same adjustment",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.1}},
						},
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.1}},
		},
		{
			name: "Multiple bid adjs, WildCard MediaType, non WildCard should have precedence",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Audio: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "static", Value: 1.0, Currency: "USD"}},
						},
					},
					WildCard: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "cpm", Value: 1.0, Currency: "USD"}},
						},
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeAudio,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustments{{AdjType: "static", Value: 1.0, Currency: "USD"}},
		},
		{
			name: "Single bid adj, Deal ID doesn't match, but wildcard present, should return given bid adj",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Native: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"*": []openrtb_ext.Adjustments{{AdjType: "cpm", Value: 1.0, Currency: "USD"}},
						},
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeNative,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustments{{AdjType: "cpm", Value: 1.0, Currency: "USD"}},
		},
		{
			name: "Single bid adj, Not matched bidder, but WildCard, should return given bid adj",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Video: map[string]map[string][]openrtb_ext.Adjustments{
						"*": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.1}},
						},
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeVideo,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.1}},
		},
		{
			name: "WildCard bidder and dealId, should return given bid adj",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					WildCard: map[string]map[string][]openrtb_ext.Adjustments{
						"*": {
							"*": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.1}},
						},
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeVideo,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.1}},
		},
		{
			name: "WildCard bidder, but dealId doesn't match given, should return nil",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					WildCard: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderB": {
							"diffDealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.1}},
						},
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeVideo,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			expected:        nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			adjArray := GetAdjustmentArray(test.givenBidAdjustments, test.givenBidType, test.givenBidderName, test.givenDealId)
			assert.Equal(t, test.expected, adjArray, "Adjustment Array doesn't match")
		})
	}
}
