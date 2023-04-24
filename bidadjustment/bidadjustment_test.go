package bidadjustment

import (
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestApplyAdjustmentArray(t *testing.T) {
	var (
		adjCur string = "EUR"
		bidCur string = "USA"
	)

	testCases := []struct {
		name             string
		givenAdjustments []openrtb_ext.Adjustment
		setMock          func(m *mock.Mock)
		givenBidPrice    float64
		expectedBidPrice float64
		expectedCurrency string
	}{
		{
			name:             "CpmAdjustment",
			givenAdjustments: []openrtb_ext.Adjustment{{Type: AdjustmentTypeCpm, Value: 1.0, Currency: adjCur}},
			givenBidPrice:    10.58687,
			setMock:          func(m *mock.Mock) { m.On("GetRate", adjCur, bidCur).Return(2.5, nil) },
			expectedBidPrice: 8.0869,
			expectedCurrency: bidCur,
		},
		{
			name:             "StaticAdjustment",
			givenAdjustments: []openrtb_ext.Adjustment{{Type: AdjustmentTypeStatic, Value: 4.0, Currency: adjCur}},
			givenBidPrice:    10.0,
			setMock:          nil,
			expectedBidPrice: 4.0,
			expectedCurrency: adjCur,
		},
		{
			name:             "MultiplierAdjustment",
			givenAdjustments: []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 3.0}},
			givenBidPrice:    10.0,
			setMock:          nil,
			expectedBidPrice: 30.0,
			expectedCurrency: bidCur,
		},
		{
			name:             "ReturnOriginalPrice",
			givenAdjustments: []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: -1.0}},
			givenBidPrice:    10.0,
			setMock:          nil,
			expectedBidPrice: 10.0,
			expectedCurrency: bidCur,
		},
		{
			name:             "NilAdjustment",
			givenAdjustments: nil,
			givenBidPrice:    10.0,
			setMock:          nil,
			expectedBidPrice: 10.0,
			expectedCurrency: bidCur,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			reqInfo := adapters.ExtraRequestInfo{}
			if test.setMock != nil {
				mockConversions := &mockConversions{}
				test.setMock(&mockConversions.Mock)
				reqInfo = adapters.NewExtraRequestInfo(mockConversions)
			}

			bidPrice, currencyAfterAdjustment := apply(test.givenAdjustments, test.givenBidPrice, bidCur, &reqInfo)
			assert.Equal(t, test.expectedBidPrice, bidPrice, "Incorrect bid prices")
			assert.Equal(t, test.expectedCurrency, currencyAfterAdjustment, "Incorrect currency")
		})
	}
}

func TestGetAndApplyAdjustmentArray(t *testing.T) {
	var (
		adjCur string = "EUR"
		bidCur string = "USA"
	)

	testCases := []struct {
		name                   string
		givenRuleToAdjustments map[string][]openrtb_ext.Adjustment
		givenBidderName        openrtb_ext.BidderName
		givenBidInfo           *adapters.TypedBid
		setMock                func(m *mock.Mock)
		expectedBidPrice       float64
		expectedCurrency       string
	}{
		{
			name: "CpmAdjustment",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  10.0,
					DealID: "dealId",
				},
				BidType: openrtb_ext.BidTypeBanner,
			},
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|bidderA|dealId": {
					{
						Type:     AdjustmentTypeCpm,
						Value:    1.0,
						Currency: adjCur,
					},
				},
				"banner|bidderA|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 2.0,
					},
				},
			},
			givenBidderName:  "bidderA",
			setMock:          func(m *mock.Mock) { m.On("GetRate", adjCur, bidCur).Return(2.5, nil) },
			expectedBidPrice: 7.5,
			expectedCurrency: bidCur,
		},
		{
			name: "StaticAdjustment",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  10.0,
					DealID: "dealId",
				},
				BidType: openrtb_ext.BidTypeBanner,
			},
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|bidderA|dealId": {
					{
						Type:     AdjustmentTypeCpm,
						Value:    1.0,
						Currency: adjCur,
					},
				},
				"banner|bidderA|*": {
					{
						Type:     AdjustmentTypeStatic,
						Value:    2.0,
						Currency: adjCur,
					},
				},
			},
			givenBidderName:  "bidderA",
			setMock:          nil,
			expectedBidPrice: 2.0,
			expectedCurrency: adjCur,
		},
		{
			name: "MultiplierAdjustment",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  10.0,
					DealID: "dealId",
				},
				BidType: openrtb_ext.BidTypeBanner,
			},
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|*|dealId": {
					{
						Type:     AdjustmentTypeCpm,
						Value:    1.0,
						Currency: adjCur,
					},
				},
				"banner|*|*": {
					{
						Type:     AdjustmentTypeMultiplier,
						Value:    2.0,
						Currency: adjCur,
					},
				},
			},
			givenBidderName:  "bidderA",
			setMock:          nil,
			expectedBidPrice: 20.0,
			expectedCurrency: bidCur,
		},
		{
			name: "NilMap",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  10.0,
					DealID: "dealId",
				},
				BidType: openrtb_ext.BidTypeBanner,
			},
			givenRuleToAdjustments: nil,
			givenBidderName:        "bidderA",
			setMock:                nil,
			expectedBidPrice:       10.0,
			expectedCurrency:       bidCur,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			reqInfo := adapters.ExtraRequestInfo{}
			if test.setMock != nil {
				mockConversions := &mockConversions{}
				test.setMock(&mockConversions.Mock)
				reqInfo = adapters.NewExtraRequestInfo(mockConversions)
			}

			bidPrice, currencyAfterAdjustment := GetAndApplyAdjustments(test.givenRuleToAdjustments, test.givenBidInfo, test.givenBidderName, bidCur, &reqInfo)
			assert.Equal(t, test.expectedBidPrice, bidPrice, "Incorrect bid prices")
			assert.Equal(t, test.expectedCurrency, currencyAfterAdjustment, "Incorrect currency")
		})
	}
}

type mockConversions struct {
	mock.Mock
}

func (m mockConversions) GetRate(from string, to string) (float64, error) {
	args := m.Called(from, to)
	return args.Get(0).(float64), args.Error(1)
}

func (m mockConversions) GetRates() *map[string]map[string]float64 {
	args := m.Called()
	return args.Get(0).(*map[string]map[string]float64)
}

func TestMergeBidAdjustments(t *testing.T) {
	testCases := []struct {
		name                   string
		givenRequestWrapper    *openrtb_ext.RequestWrapper
		givenAccount           *config.Account
		expectedBidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments
	}{
		{
			name: "DiffBidderNames",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"banner":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: openrtb_ext.MediaType{
						Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
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
			name: "RequestTakesPrecedence",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"audio":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: openrtb_ext.MediaType{
						Audio: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
							"bidderA": {
								"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Audio: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.1}},
						},
					},
				},
			},
		},
		{
			name: "DiffDealIds",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"video":{"bidderA":{"dealId":[{ "adjtype": "static", "value": 3.00, "currency": "USD"}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: openrtb_ext.MediaType{
						Video: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
							"bidderA": {
								"diffDealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Video: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId":     []openrtb_ext.Adjustment{{Type: "static", Value: 3.00, Currency: "USD"}},
							"diffDealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
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
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: openrtb_ext.MediaType{
						Native: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Native: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: "cpm", Value: 0.18, Currency: "USD"}},
						},
						"bidderB": {
							"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
						},
					},
				},
			},
		},
		{
			name: "ReqAdjVideoAcctAdjBanner",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"video":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: openrtb_ext.MediaType{
						Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderB": {
							"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
						},
					},
					Video: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.1}},
						},
					},
				},
			},
		},
		{
			name: "RequestNilPrebid",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"ext":{"bidder": {}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: openrtb_ext.MediaType{
						Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderB": {
							"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
						},
					},
				},
			},
		},
		{
			name: "AcctWildCardRequestVideo",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"video":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: openrtb_ext.MediaType{
						WildCard: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					WildCard: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderB": {
							"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
						},
					},
					Video: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.1}},
						},
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mergedBidAdj, err := merge(test.givenRequestWrapper, test.givenAccount.BidAdjustments)
			assert.NoError(t, err, "Unexpected error received")
			assert.Equal(t, test.expectedBidAdjustments, mergedBidAdj)
		})
	}
}

func TestGenerateMap(t *testing.T) {
	testCases := []struct {
		name                string
		givenBidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments
		expectedMap         map[string][]openrtb_ext.Adjustment
	}{
		{
			name: "OneAdjustment",
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.1}},
						},
					},
				},
			},
			expectedMap: map[string][]openrtb_ext.Adjustment{
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
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}},
						},
						"*": {
							"diffDealId": []openrtb_ext.Adjustment{{Type: AdjustmentTypeCpm, Value: 1.1, Currency: "USD"}},
							"*":          []openrtb_ext.Adjustment{{Type: AdjustmentTypeStatic, Value: 5.0, Currency: "USD"}},
						},
					},
					Video: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
						"*": {
							"*": []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}, {Type: AdjustmentTypeCpm, Value: 0.18, Currency: "USD"}},
						},
					},
				},
			},
			expectedMap: map[string][]openrtb_ext.Adjustment{
				"banner|bidderA|dealId": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
				"banner|*|diffDealId": {
					{
						Type:     AdjustmentTypeCpm,
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
				"video|*|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
					{
						Type:     AdjustmentTypeCpm,
						Value:    0.18,
						Currency: "USD",
					},
				},
			},
		},
		{
			name:                "NilAdjustments",
			givenBidAdjustments: nil,
			expectedMap:         nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ruleToAdjustmentMap := GenerateMap(test.givenBidAdjustments)
			assert.Equal(t, test.expectedMap, ruleToAdjustmentMap)
		})
	}
}

func TestProcessBidAdjustments(t *testing.T) {
	testCases := []struct {
		name                   string
		givenRequestWrapper    *openrtb_ext.RequestWrapper
		givenAccount           *config.Account
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
						Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
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
						Banner: map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mergedBidAdj, err := Process(test.givenRequestWrapper, test.givenAccount.BidAdjustments)
			assert.NoError(t, err, "Unexpected error received")
			assert.Equal(t, test.expectedBidAdjustments, mergedBidAdj)
		})
	}
}
