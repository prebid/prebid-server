package bidadjustment

import (
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name                string
		givenBidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments
		expected            bool
	}{
		{
			name: "OneAdjustmentValid",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "MultipleAdjustmentsValid",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Audio: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}},
						},
					},
					VideoInstream: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeStatic, Value: 3.0, Currency: "USD"}},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "MixOfValidandInvalid",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Native: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}},
						},
					},
					VideoOutstream: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: ""}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "WildCardInvalid",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					WildCard: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: -1.1, Currency: "USD"}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "AudioInvalid",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Audio: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeStatic, Value: 5.0, Currency: ""}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "NativeInvalid",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Native: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: -1.1, Currency: "USD"}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "BannerInvalid",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 150}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "InstreamInvalid",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					VideoInstream: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 150}},
						},
					},
				},
			},
			expected: false,
		},
		{
			name:                "EmptyBidAdjustments",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{},
			expected:            true,
		},
		{
			name:                "NilBidAdjustments",
			givenBidAdjustments: nil,
			expected:            true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actual := Validate(test.givenBidAdjustments)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestValidateForMediaType(t *testing.T) {
	testCases := []struct {
		name        string
		givenBidAdj map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID
		expected    bool
	}{
		{
			name: "OneAdjustmentValid",
			givenBidAdj: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
				"bidderA": {
					"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}},
				},
			},
			expected: true,
		},
		{
			name: "OneAdjustmentInvalid",
			givenBidAdj: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
				"bidderA": {
					"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: -1.1}},
				},
			},
			expected: false,
		},
		{
			name: "MultipleAdjustmentsValid",
			givenBidAdj: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
				"bidderA": {
					"dealId": []openrtb_ext.Adjustment{
						{Type: AdjustmentTypeMultiplier, Value: 1.1},
						{Type: AdjustmentTypeStatic, Value: 3.0, Currency: "USD"},
					},
				},
			},
			expected: true,
		},
		{
			name: "MultipleAdjustmentsInvalid",
			givenBidAdj: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
				"bidderA": {
					"dealId": []openrtb_ext.Adjustment{
						{Type: AdjustmentTypeMultiplier, Value: -1.1},
						{Type: AdjustmentTypeCPM, Value: -3.0, Currency: "USD"},
					},
				},
			},
			expected: false,
		},
		{
			name: "MultipleDealIdsValid",
			givenBidAdj: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
				"bidderA": {
					"dealId": []openrtb_ext.Adjustment{
						{Type: AdjustmentTypeStatic, Value: 3.0, Currency: "USD"},
					},
					"diffDealId": []openrtb_ext.Adjustment{
						{Type: AdjustmentTypeMultiplier, Value: 1.1},
					},
				},
			},
			expected: true,
		},
		{
			name: "MultipleBiddersValid",
			givenBidAdj: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
				"bidderA": {
					"dealId": []openrtb_ext.Adjustment{
						{Type: AdjustmentTypeStatic, Value: 5.0, Currency: "USD"},
					},
				},
				"bidderB": {
					"dealId": []openrtb_ext.Adjustment{
						{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"},
					},
				},
			},
			expected: true,
		},
		{
			name:        "NilBidAdj",
			givenBidAdj: nil,
			expected:    true,
		},
		{
			name: "NilBidderToAdjustmentsByDealID",
			givenBidAdj: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
				"bidderA": nil,
			},
			expected: false,
		},
		{
			name: "NilDealIdToAdjustments",
			givenBidAdj: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
				"bidderA": {
					"dealId": nil,
				},
			},
			expected: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actual := validateForMediaType(test.givenBidAdj)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestValidateAdjustment(t *testing.T) {
	testCases := []struct {
		name            string
		givenAdjustment openrtb_ext.Adjustment
		expected        bool
	}{
		{
			name: "ValidCpm",
			givenAdjustment: openrtb_ext.Adjustment{
				Type:     AdjustmentTypeCPM,
				Value:    5.0,
				Currency: "USD",
			},
			expected: true,
		},
		{
			name: "ValidMultiplier",
			givenAdjustment: openrtb_ext.Adjustment{
				Type:  AdjustmentTypeMultiplier,
				Value: 2.0,
			},
			expected: true,
		},
		{
			name: "ValidStatic",
			givenAdjustment: openrtb_ext.Adjustment{
				Type:     AdjustmentTypeStatic,
				Value:    3.0,
				Currency: "USD",
			},
			expected: true,
		},
		{
			name: "InvalidCpm",
			givenAdjustment: openrtb_ext.Adjustment{
				Type:     AdjustmentTypeCPM,
				Value:    5.0,
				Currency: "",
			},
			expected: false,
		},
		{
			name: "InvalidMultiplier",
			givenAdjustment: openrtb_ext.Adjustment{
				Type:  AdjustmentTypeMultiplier,
				Value: 200,
			},
			expected: false,
		},
		{
			name: "InvalidStatic",
			givenAdjustment: openrtb_ext.Adjustment{
				Type:     AdjustmentTypeStatic,
				Value:    -3.0,
				Currency: "USD",
			},
			expected: false,
		},
		{
			name: "InvalidAdjType",
			givenAdjustment: openrtb_ext.Adjustment{
				Type:  "Invalid",
				Value: 1.0,
			},
			expected: false,
		},
		{
			name:            "EmptyAdjustment",
			givenAdjustment: openrtb_ext.Adjustment{},
			expected:        false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actual := validateAdjustment(test.givenAdjustment)
			assert.Equal(t, test.expected, actual)
		})
	}
}
