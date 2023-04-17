package exchange

import (
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestApplyAdjustmentArray(t *testing.T) {
	var (
		givenFrom string = "EUR"
		givenTo   string = "USA"
	)

	testCases := []struct {
		name             string
		givenAdjustments []openrtb_ext.Adjustments
		setMock          func(m *mock.Mock)
		givenBidPrice    float64
		expectedBidPrice float64
	}{
		{
			name:             "CPM adj type, value after currency conversion should be subtracted from given bid price. Price should round to 4 decimal places",
			givenAdjustments: []openrtb_ext.Adjustments{{AdjType: openrtb_ext.AdjTypeCpm, Value: 1.0, Currency: &givenTo}},
			givenBidPrice:    10.58687,
			setMock:          func(m *mock.Mock) { m.On("GetRate", "EUR", "USA").Return(2.5, nil) },
			expectedBidPrice: 8.0869,
		},
		{
			name:             "Static adj type, value after currency conversion should be the bid price",
			givenAdjustments: []openrtb_ext.Adjustments{{AdjType: openrtb_ext.AdjTypeStatic, Value: 4.0, Currency: &givenTo}},
			givenBidPrice:    10.0,
			setMock:          func(m *mock.Mock) { m.On("GetRate", "EUR", "USA").Return(5.0, nil) },
			expectedBidPrice: 20.0,
		},
		{
			name:             "Multiplier adj type with no currency conversion",
			givenAdjustments: []openrtb_ext.Adjustments{{AdjType: openrtb_ext.AdjTypeMultiplier, Value: 3.0}},
			givenBidPrice:    10.0,
			expectedBidPrice: 30.0,
		},
		{
			name:             "Bid price after conversions is equal or less than 0, should return original bid price instead",
			givenAdjustments: []openrtb_ext.Adjustments{{AdjType: openrtb_ext.AdjTypeMultiplier, Value: -1.0}},
			givenBidPrice:    10.0,
			expectedBidPrice: 10.0,
		},
		{
			name:             "Nil adjustment array",
			givenAdjustments: nil,
			givenBidPrice:    10.0,
			expectedBidPrice: 10.0,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			reqInfo := adapters.ExtraRequestInfo{}
			if test.givenAdjustments != nil && test.givenAdjustments[0].Currency != nil {
				mockConversions := &mockConversions{}
				test.setMock(&mockConversions.Mock)
				reqInfo = adapters.NewExtraRequestInfo(mockConversions)
			}

			bidPrice := applyAdjustmentArray(test.givenAdjustments, test.givenBidPrice, givenFrom, &reqInfo)
			assert.Equal(t, test.expectedBidPrice, bidPrice, "Incorrect bid prices")
		})
	}
}

func TestGetAndApplyAdjustmentArray(t *testing.T) {
	var (
		givenFrom string = "EUR"
		givenTo   string = "USA"
	)

	testCases := []struct {
		name                string
		givenBidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments
		givenBidderName     openrtb_ext.BidderName
		givenBidInfo        *adapters.TypedBid
		setMock             func(m *mock.Mock)
		expectedBidPrice    float64
	}{
		{
			name: "Valid Bid Adjustments, CPM adj type, function should Get and Apply the adjustment properly",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  10.0,
					DealID: "dealId",
				},
				BidType: openrtb_ext.BidTypeBanner,
			},
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: &openrtb_ext.MediaType{
					Banner: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: openrtb_ext.AdjTypeCpm, Value: 1.0, Currency: &givenTo}},
						},
					},
				},
			},
			givenBidderName:  "bidderA",
			setMock:          func(m *mock.Mock) { m.On("GetRate", "EUR", "USA").Return(2.5, nil) },
			expectedBidPrice: 7.5,
		},
		{
			name: "Nil adjustment array, expect no change to the bid price",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price: 10.0,
				},
				BidType: openrtb_ext.BidTypeBanner,
			},
			givenBidAdjustments: nil,
			expectedBidPrice:    10.0,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			reqInfo := adapters.ExtraRequestInfo{}
			if test.givenBidAdjustments != nil {
				mockConversions := &mockConversions{}
				test.setMock(&mockConversions.Mock)
				reqInfo = adapters.NewExtraRequestInfo(mockConversions)
			}

			bidPrice := getAndApplyAdjustmentArray(test.givenBidAdjustments, test.givenBidInfo, test.givenBidderName, givenFrom, &reqInfo)
			assert.Equal(t, test.expectedBidPrice, bidPrice, "Incorrect bid prices")
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
	currency := "USD"

	testCases := []struct {
		name                   string
		givenRequestWrapper    *openrtb_ext.RequestWrapper
		givenAccount           *config.Account
		expectedBidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments
	}{
		{
			name: "Different Bidder Names for Bid Adjustments Present in Request and Account",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"banner":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: &openrtb_ext.MediaType{
						Banner: map[string]map[string][]openrtb_ext.Adjustments{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: &openrtb_ext.MediaType{
					Banner: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.1}},
						},
						"bidderB": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
						},
					},
				},
			},
		},
		{
			name: "Same Bidder Name and DealIDs for Bid Adjustments Present in Request and Account. Request should take precedence",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"audio":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: &openrtb_ext.MediaType{
						Audio: map[string]map[string][]openrtb_ext.Adjustments{
							"bidderA": {
								"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: &openrtb_ext.MediaType{
					Audio: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.1}},
						},
					},
				},
			},
		},
		{
			name: "Same Bidder Name, different DealIDs for Bid Adjustments Present in Request and Account.",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"video":{"bidderA":{"dealId":[{ "adjtype": "static", "value": 3.00, "currency": "USD"}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: &openrtb_ext.MediaType{
						Video: map[string]map[string][]openrtb_ext.Adjustments{
							"bidderA": {
								"diffDealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: &openrtb_ext.MediaType{
					Video: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId":     []openrtb_ext.Adjustments{{AdjType: "static", Value: 3.00, Currency: &currency}},
							"diffDealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
						},
					},
				},
			},
		},
		{
			name: "Different bidder names, request comes with CPM bid adjustment",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"native":{"bidderA":{"dealId":[{"adjtype": "cpm", "value": 0.18, "currency": "USD"}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: &openrtb_ext.MediaType{
						Native: map[string]map[string][]openrtb_ext.Adjustments{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: &openrtb_ext.MediaType{
					Native: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "cpm", Value: 0.18, Currency: &currency}},
						},
						"bidderB": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
						},
					},
				},
			},
		},
		{
			name: "Account has Banner Adjustment, Request has Video Adjustment",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"video":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: &openrtb_ext.MediaType{
						Banner: map[string]map[string][]openrtb_ext.Adjustments{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: &openrtb_ext.MediaType{
					Banner: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderB": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
						},
					},
					Video: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.1}},
						},
					},
				},
			},
		},
		{
			name: "Request has nil ExtPrebid, Account has Banner Adjustment",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"ext":{"bidder": {}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: &openrtb_ext.MediaType{
						Banner: map[string]map[string][]openrtb_ext.Adjustments{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: &openrtb_ext.MediaType{
					Banner: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderB": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
						},
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mergedBidAdj, err := mergeBidAdjustments(test.givenRequestWrapper, test.givenAccount.BidAdjustments)
			assert.NoError(t, err, "Unexpected error received")
			assert.Equal(t, test.expectedBidAdjustments, mergedBidAdj)
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
			name: "Valid Request and Account Adjustments with different bidder names, should properly merge",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"banner":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: &openrtb_ext.MediaType{
						Banner: map[string]map[string][]openrtb_ext.Adjustments{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: &openrtb_ext.MediaType{
					Banner: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.1}},
						},
						"bidderB": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
						},
					},
				},
			},
		},
		{
			name: "Invalid Request Adjustment, Expect Nil Merged Adjustments",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"banner":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 200}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: &openrtb_ext.MediaType{
						Banner: map[string]map[string][]openrtb_ext.Adjustments{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
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
			mergedBidAdj, err := processBidAdjustments(test.givenRequestWrapper, test.givenAccount.BidAdjustments)
			assert.NoError(t, err, "Unexpected error received")
			assert.Equal(t, test.expectedBidAdjustments, mergedBidAdj)
		})
	}
}
