package bidadjustment

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetAndApply(t *testing.T) {
	var (
		adjCur string = "EUR"
		bidCur string = "USA"
	)

	testCases := []struct {
		name                   string
		givenRuleToAdjustments map[string][]openrtb_ext.Adjustment
		givenBidderName        openrtb_ext.BidderName
		givenBidInfo           *adapters.TypedBid
		givenBidType           string
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
			},
			givenBidType: string(openrtb_ext.BidTypeBanner),
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|bidderA|dealId": {
					{
						Type:     AdjustmentTypeCPM,
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
			},
			givenBidType: string(openrtb_ext.BidTypeBanner),
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|bidderA|dealId": {
					{
						Type:     AdjustmentTypeCPM,
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
			},
			givenBidType: VideoInstream,
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|*|dealId": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    1.0,
						Currency: adjCur,
					},
				},
				"video-instream|*|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 2.0,
					},
				},
			},
			givenBidderName:  "bidderA",
			setMock:          nil,
			expectedBidPrice: 20.0,
			expectedCurrency: bidCur,
		},
		{
			name: "CpmAndMultiplierAdjustments",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  10.0,
					DealID: "dealId",
				},
			},
			givenBidType: VideoInstream,
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"video-instream|*|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 2.0,
					},
					{
						Type:     AdjustmentTypeCPM,
						Value:    1.0,
						Currency: adjCur,
					},
				},
			},
			givenBidderName:  "bidderA",
			setMock:          func(m *mock.Mock) { m.On("GetRate", adjCur, bidCur).Return(2.5, nil) },
			expectedBidPrice: 17.5,
			expectedCurrency: bidCur,
		},
		{
			name: "DealIdPresentAndNegativeAdjustedPrice",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  1.0,
					DealID: "dealId",
				},
			},
			givenBidType: VideoInstream,
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|*|dealId": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    1.0,
						Currency: adjCur,
					},
				},
			},
			givenBidderName:  "bidderA",
			setMock:          func(m *mock.Mock) { m.On("GetRate", adjCur, bidCur).Return(2.5, nil) },
			expectedBidPrice: 0.0,
			expectedCurrency: bidCur,
		},
		{
			name: "NoDealIdNegativeAdjustedPrice",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  1.0,
					DealID: "",
				},
			},
			givenBidType: string(openrtb_ext.BidTypeAudio),
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|*|*": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    1.0,
						Currency: adjCur,
					},
				},
			},
			givenBidderName:  "bidderA",
			setMock:          func(m *mock.Mock) { m.On("GetRate", adjCur, bidCur).Return(2.5, nil) },
			expectedBidPrice: 0.1,
			expectedCurrency: bidCur,
		},
		{
			name: "NilMap",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  10.0,
					DealID: "dealId",
				},
			},
			givenBidType:           string(openrtb_ext.BidTypeBanner),
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
			bidPrice, currencyAfterAdjustment := Apply(test.givenRuleToAdjustments, test.givenBidInfo, test.givenBidderName, bidCur, &reqInfo, test.givenBidType)
			assert.Equal(t, test.expectedBidPrice, bidPrice)
			assert.Equal(t, test.expectedCurrency, currencyAfterAdjustment)
		})
	}
}

type mockConversions struct {
	mock.Mock
}

func (m *mockConversions) GetRate(from string, to string) (float64, error) {
	args := m.Called(from, to)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockConversions) GetRates() *map[string]map[string]float64 {
	args := m.Called()
	return args.Get(0).(*map[string]map[string]float64)
}

func TestApply(t *testing.T) {
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
			givenAdjustments: []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 1.0, Currency: adjCur}},
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
			assert.Equal(t, test.expectedBidPrice, bidPrice)
			assert.Equal(t, test.expectedCurrency, currencyAfterAdjustment)
		})
	}
}

func TestGet(t *testing.T) {
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
						Type:     AdjustmentTypeCPM,
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
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority4",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|bidderA|dealId": {
					{
						Type:     AdjustmentTypeCPM,
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
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority5",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|*|*": {
					{
						Type:     AdjustmentTypeCPM,
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
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority6",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|bidderA|*": {
					{
						Type:     AdjustmentTypeCPM,
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
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority7",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|*|dealId": {
					{
						Type:     AdjustmentTypeCPM,
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
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
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
		{
			name: "NoPriorityRulesMatch",
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
			givenBidType:    openrtb_ext.BidTypeVideo,
			givenBidderName: "bidderB",
			givenDealId:     "diffDealId",
			expected:        nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			adjArray := get(test.givenRuleToAdjustments, string(test.givenBidType), string(test.givenBidderName), test.givenDealId)
			assert.Equal(t, test.expected, adjArray)
		})
	}
}
