package bidadjustments

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
			name: "Valid single bid adjustment multiplier",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.1}},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Valid banner bid adjustment, invalid video bid adjustment, negative value",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.1}},
						},
					},
					Video: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: -1.0}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Invalid bid adjustment value, too large",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Audio: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 200}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Valid bid adjustment cpm",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Native: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "cpm", Value: 1.0, Currency: "USD"}},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Invalid CPM bid adjustment, no currency given",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "cpm", Value: 1.0, Currency: ""}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Invalid CPM bid adjustment, negative value",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Native: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "cpm", Value: -1.0, Currency: "USD"}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Valid static bid adjustment",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "static", Value: 1.0, Currency: "USD"}},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Invalid static bid adjustment, no currency",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "static", Value: 1.0, Currency: ""}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Invalid static bid adjustment, negative value",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "static", Value: -1.0, Currency: "USD"}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Invalid wildcard adjustment, negative value",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					WildCard: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "static", Value: -1.0, Currency: "USD"}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name:                "Nil Bid Adjustment",
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
