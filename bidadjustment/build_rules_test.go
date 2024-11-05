package bidadjustment

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestBuildRules(t *testing.T) {
	testCases := []struct {
		name                string
		givenBidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments
		expectedRules       map[string][]openrtb_ext.Adjustment
	}{
		{
			name: "OneAdjustment",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}},
						},
					},
				},
			},
			expectedRules: map[string][]openrtb_ext.Adjustment{
				"banner|bidderA|dealId": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
		},
		{
			name: "MultipleAdjustments",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}},
						},
						"*": {
							"diffDealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 1.1, Currency: "USD"}},
							"*":          []openrtb_ext.Adjustment{{Type: AdjustmentTypeStatic, Value: 5.0, Currency: "USD"}},
						},
					},
					VideoInstream: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"*": {
							"*": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}, {Type: AdjustmentTypeCPM, Value: 0.18, Currency: "USD"}},
						},
					},
					VideoOutstream: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderB": {
							"*": []openrtb_ext.Adjustment{{Type: AdjustmentTypeStatic, Value: 0.25, Currency: "USD"}},
						},
					},
				},
			},
			expectedRules: map[string][]openrtb_ext.Adjustment{
				"banner|bidderA|dealId": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
				"banner|*|diffDealId": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    1.1,
						Currency: "USD",
					},
				},
				"banner|*|*": {
					{
						Type:     AdjustmentTypeStatic,
						Value:    5.0,
						Currency: "USD",
					},
				},
				"video-instream|*|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
					{
						Type:     AdjustmentTypeCPM,
						Value:    0.18,
						Currency: "USD",
					},
				},
				"video-outstream|bidderB|*": {
					{
						Type:     AdjustmentTypeStatic,
						Value:    0.25,
						Currency: "USD",
					},
				},
			},
		},
		{
			name:                "NilAdjustments",
			givenBidAdjustments: nil,
			expectedRules:       nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			rules := BuildRules(test.givenBidAdjustments)
			assert.Equal(t, test.expectedRules, rules)
		})
	}
}

func TestMergeAndValidate(t *testing.T) {
	testCases := []struct {
		name                   string
		givenRequestWrapper    *openrtb_ext.RequestWrapper
		givenAccount           *config.Account
		expectError            bool
		expectedBidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments
	}{
		{
			name: "ValidReqAndAcctAdjustments",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"banner":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: openrtb_ext.MediaType{
						Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectError: false,
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.1}},
						},
						"bidderB": {
							"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
						},
					},
				},
			},
		},
		{
			name: "InvalidReqAdjustment",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"banner":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 200}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: openrtb_ext.MediaType{
						Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectError:            true,
			expectedBidAdjustments: nil,
		},
		{
			name: "InvalidAcctAdjustment",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"banner":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: openrtb_ext.MediaType{
						Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: -1.5}},
							},
						},
					},
				},
			},
			expectError:            true,
			expectedBidAdjustments: nil,
		},
		{
			name: "InvalidJSON",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{}}`)},
			},
			givenAccount:           &config.Account{},
			expectError:            true,
			expectedBidAdjustments: nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mergedBidAdj, err := Merge(test.givenRequestWrapper, test.givenAccount.BidAdjustments)
			if !test.expectError {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			assert.Equal(t, test.expectedBidAdjustments, mergedBidAdj)
		})
	}
}

func TestMerge(t *testing.T) {
	testCases := []struct {
		name                   string
		givenRequestWrapper    *openrtb_ext.RequestWrapper
		acctBidAdjustments     *openrtb_ext.ExtRequestPrebidBidAdjustments
		expectedBidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments
	}{
		{
			name: "DiffBidderNames",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"banner":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			acctBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderB": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.5}}},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}}},
						"bidderB": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.5}}},
					},
				},
			},
		},
		{
			name: "RequestTakesPrecedence",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"audio":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			acctBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Audio: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.5}}},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Audio: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}}},
					},
				},
			},
		},
		{
			name: "DiffDealIds",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"video-instream":{"bidderA":{"dealId":[{ "adjtype": "static", "value": 3.00, "currency": "USD"}]}}}}}}`)},
			},
			acctBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					VideoInstream: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {"diffDealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.5}}},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					VideoInstream: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {
							"dealId":     []openrtb_ext.Adjustment{{Type: AdjustmentTypeStatic, Value: 3.00, Currency: "USD"}},
							"diffDealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.5}},
						},
					},
				},
			},
		},
		{
			name: "DiffBidderNamesCpm",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"native":{"bidderA":{"dealId":[{"adjtype": "cpm", "value": 0.18, "currency": "USD"}]}}}}}}`)},
			},
			acctBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Native: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderB": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.5}}},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Native: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 0.18, Currency: "USD"}}},
						"bidderB": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.5}}},
					},
				},
			},
		},
		{
			name: "ReqAdjVideoAcctAdjBanner",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"video-outstream":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			acctBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderB": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.5}}},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderB": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.5}}},
					},
					VideoOutstream: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}}},
					},
				},
			},
		},
		{
			name: "RequestNilPrebid",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"ext":{"bidder": {}}}`)},
			},
			acctBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderB": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.5}}},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderB": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.5}}},
					},
				},
			},
		},
		{
			name: "AcctWildCardRequestVideo",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"video-instream":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			acctBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					WildCard: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderB": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.5}}},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					WildCard: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderB": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.5}}},
					},
					VideoInstream: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}}},
					},
				},
			},
		},
		{
			name: "NilReqExtPrebidAndAcctBidAdj",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"ext":{"bidder": {}}}`)},
			},
			acctBidAdjustments:     nil,
			expectedBidAdjustments: nil,
		},
		{
			name: "NilAcctBidAdj",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"banner":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			acctBidAdjustments: nil,
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}}},
					},
				},
			},
		},

		{
			name: "NilExtPrebid-NilExtPrebidBidAdj_NilAcct",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			acctBidAdjustments:     nil,
			expectedBidAdjustments: nil,
		},
		{
			name: "NilExtPrebid-NilExtPrebidBidAdj-Acct",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			acctBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}}},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}}},
					},
				},
			},
		},
		{
			name: "NotNilExtPrebid-NilExtBidAdj-NilAcct",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{}}`)},
			},
			acctBidAdjustments:     nil,
			expectedBidAdjustments: nil,
		},
		{
			name: "NotNilExtPrebid_NilExtBidAdj_NotNilAcct",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{}}`)},
			},
			acctBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}}},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}}},
					},
				},
			},
		},
		{
			name: "NotNilExtPrebid-NotNilExtBidAdj-NilAcct",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"banner":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			acctBidAdjustments: nil,
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}}},
					},
				},
			},
		},
		{
			name: "NotNilExtPrebid-NotNilExtBidAdj-NotNilAcct",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"banner":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			acctBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					VideoInstream: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderB": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3}}},
					},
					VideoOutstream: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderC": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3}}},
					},
					Audio: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderD": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3}}},
					},
					Native: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderE": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3}}},
					},
					WildCard: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderF": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3}}},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderA": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}}},
					},
					VideoInstream: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderB": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3}}},
					},
					VideoOutstream: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderC": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3}}},
					},
					Audio: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderD": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3}}},
					},
					Native: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderE": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3}}},
					},
					WildCard: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
						"bidderF": {"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3}}},
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mergedBidAdj, err := merge(test.givenRequestWrapper, test.acctBidAdjustments)
			assert.NoError(t, err)
			assert.Equal(t, test.expectedBidAdjustments, mergedBidAdj)
		})
	}
}
