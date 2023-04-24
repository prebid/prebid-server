package bidadjustment

import (
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestValidateBidAdjustments(t *testing.T) {
	testCases := []struct {
		name                string
		givenBidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments
		expected            bool
	}{
		{
			name: "ValidMultiplierAdjustment",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "InvalidAdjustmentNegative",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}},
						},
					},
					Video: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: -1.0}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "InvalidAdjustmentTooLarge",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Audio: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 200}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "ValidCpmAdjustment",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Native: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCpm, Value: 1.0, Currency: "USD"}},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "InvalidCpmAdjustmentNoCurrency",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCpm, Value: 1.0, Currency: ""}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "InvalidAdjustmentNegative",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Native: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCpm, Value: -1.0, Currency: "USD"}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "ValidStaticAdjustment",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: "static", Value: 1.0, Currency: "USD"}},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "InvalidStaticAdjustmentNoCurrency",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: "static", Value: 1.0, Currency: ""}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "InvalidStaticAdjustmentNegative",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: "static", Value: -1.0, Currency: "USD"}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "InvalidWildcardAdjustmentNegative",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					WildCard: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: "static", Value: -1.0, Currency: "USD"}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name:                "NilAdjustment",
			givenBidAdjustments: nil,
			expected:            true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actual := Validate(test.givenBidAdjustments)
			assert.Equal(t, test.expected, actual, "Boolean didn't match")
		})
	}
}
