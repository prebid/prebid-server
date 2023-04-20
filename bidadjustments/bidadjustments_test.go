package bidadjustments

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
		givenAdjustments []openrtb_ext.Adjustments
		setMock          func(m *mock.Mock)
		givenBidPrice    float64
		expectedBidPrice float64
		expectedCurrency string
	}{
		{
			name:             "CPM adj type, value after currency conversion should be subtracted from given bid price. Price should round to 4 decimal places",
			givenAdjustments: []openrtb_ext.Adjustments{{AdjType: AdjTypeCpm, Value: 1.0, Currency: adjCur}},
			givenBidPrice:    10.58687,
			setMock:          func(m *mock.Mock) { m.On("GetRate", adjCur, bidCur).Return(2.5, nil) },
			expectedBidPrice: 8.0869,
			expectedCurrency: bidCur,
		},
		{
			name:             "Static adj type, that static value should be the bid price, currency should be updated as well",
			givenAdjustments: []openrtb_ext.Adjustments{{AdjType: AdjTypeStatic, Value: 4.0, Currency: adjCur}},
			givenBidPrice:    10.0,
			setMock:          nil,
			expectedBidPrice: 4.0,
			expectedCurrency: adjCur,
		},
		{
			name:             "Multiplier adj type with no currency conversion",
			givenAdjustments: []openrtb_ext.Adjustments{{AdjType: AdjTypeMultiplier, Value: 3.0}},
			givenBidPrice:    10.0,
			setMock:          nil,
			expectedBidPrice: 30.0,
			expectedCurrency: bidCur,
		},
		{
			name:             "Bid price after conversions is equal or less than 0, should return original bid price instead",
			givenAdjustments: []openrtb_ext.Adjustments{{AdjType: AdjTypeMultiplier, Value: -1.0}},
			givenBidPrice:    10.0,
			setMock:          nil,
			expectedBidPrice: 10.0,
			expectedCurrency: bidCur,
		},
		{
			name:             "Nil adjustment array",
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

			bidPrice, currencyAfterAdjustment := applyAdjustmentArray(test.givenAdjustments, test.givenBidPrice, bidCur, &reqInfo)
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
		name                string
		givenBidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments
		givenBidderName     openrtb_ext.BidderName
		givenBidInfo        *adapters.TypedBid
		setMock             func(m *mock.Mock)
		expectedBidPrice    float64
		expectedCurrency    string
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
				MediaType: openrtb_ext.MediaType{
					Banner: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: AdjTypeCpm, Value: 1.0, Currency: adjCur}},
						},
					},
				},
			},
			givenBidderName:  "bidderA",
			setMock:          func(m *mock.Mock) { m.On("GetRate", adjCur, bidCur).Return(2.5, nil) },
			expectedBidPrice: 7.5,
			expectedCurrency: bidCur,
		},
		{
			name: "Valid Bid Adjustments, static adj type, function should Get and Apply the adjustment properly",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  10.0,
					DealID: "dealId",
				},
				BidType: openrtb_ext.BidTypeBanner,
			},
			givenBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Banner: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: AdjTypeStatic, Value: 5.0, Currency: adjCur}},
						},
					},
				},
			},
			givenBidderName:  "bidderA",
			setMock:          nil,
			expectedBidPrice: 5.0,
			expectedCurrency: adjCur,
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
			expectedCurrency:    bidCur,
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

			bidPrice, currencyAfterAdjustment := GetAndApplyAdjustmentArray(test.givenBidAdjustments, test.givenBidInfo, test.givenBidderName, bidCur, &reqInfo)
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
			name: "Different Bidder Names for Bid Adjustments Present in Request and Account",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidadjustments":{"mediatype":{"banner":{"bidderA":{"dealId":[{ "adjtype": "multiplier", "value": 1.1}]}}}}}}`)},
			},
			givenAccount: &config.Account{
				BidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
					MediaType: openrtb_ext.MediaType{
						Banner: map[string]map[string][]openrtb_ext.Adjustments{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
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
					MediaType: openrtb_ext.MediaType{
						Audio: map[string]map[string][]openrtb_ext.Adjustments{
							"bidderA": {
								"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
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
					MediaType: openrtb_ext.MediaType{
						Video: map[string]map[string][]openrtb_ext.Adjustments{
							"bidderA": {
								"diffDealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Video: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId":     []openrtb_ext.Adjustments{{AdjType: "static", Value: 3.00, Currency: "USD"}},
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
					MediaType: openrtb_ext.MediaType{
						Native: map[string]map[string][]openrtb_ext.Adjustments{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
					Native: map[string]map[string][]openrtb_ext.Adjustments{
						"bidderA": {
							"dealId": []openrtb_ext.Adjustments{{AdjType: "cpm", Value: 0.18, Currency: "USD"}},
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
					MediaType: openrtb_ext.MediaType{
						Banner: map[string]map[string][]openrtb_ext.Adjustments{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
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
					MediaType: openrtb_ext.MediaType{
						Banner: map[string]map[string][]openrtb_ext.Adjustments{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
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
					MediaType: openrtb_ext.MediaType{
						Banner: map[string]map[string][]openrtb_ext.Adjustments{
							"bidderB": {
								"dealId": []openrtb_ext.Adjustments{{AdjType: "multiplier", Value: 1.5}},
							},
						},
					},
				},
			},
			expectedBidAdjustments: &openrtb_ext.ExtRequestPrebidBidAdjustments{
				MediaType: openrtb_ext.MediaType{
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
					MediaType: openrtb_ext.MediaType{
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
			mergedBidAdj, err := ProcessBidAdjustments(test.givenRequestWrapper, test.givenAccount.BidAdjustments)
			assert.NoError(t, err, "Unexpected error received")
			assert.Equal(t, test.expectedBidAdjustments, mergedBidAdj)
		})
	}
}
