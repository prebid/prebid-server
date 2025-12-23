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
			name: "CpmAdjustment no seat",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  10.0,
					DealID: "dealId",
				},
			},
			givenBidType: string(openrtb_ext.BidTypeBanner),
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|biddera|dealid": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    1.0,
						Currency: adjCur,
					},
				},
				"banner|biddera|*": {
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
			name: "CpmAdjustment with seat",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  10.0,
					DealID: "dealId",
				},
				Seat: "seatA",
			},
			givenBidType: string(openrtb_ext.BidTypeBanner),
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|seata|dealid": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    1.0,
						Currency: adjCur,
					},
				},
				"banner|seata|*": {
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
			name: "StaticAdjustment no seat",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  10.0,
					DealID: "dealId",
				},
			},
			givenBidType: string(openrtb_ext.BidTypeBanner),
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|biddera|dealid": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    1.0,
						Currency: adjCur,
					},
				},
				"banner|biddera|*": {
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
			name: "StaticAdjustment with seat",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  10.0,
					DealID: "dealId",
				},
				Seat: "seatA",
			},
			givenBidType: string(openrtb_ext.BidTypeBanner),
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|seata|dealid": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    1.0,
						Currency: adjCur,
					},
				},
				"banner|seata|*": {
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
			name: "MultiplierAdjustment no seat",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  10.0,
					DealID: "dealId",
				},
			},
			givenBidType: VideoInstream,
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|*|dealid": {
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
			name: "MultiplierAdjustment with seat",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  10.0,
					DealID: "dealId",
				},
				Seat: "seatA",
			},
			givenBidType: VideoInstream,
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|*|dealid": {
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
			name: "CpmAndMultiplierAdjustments no seat",
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
			name: "CpmAndMultiplierAdjustments with seat",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  10.0,
					DealID: "dealId",
				},
				Seat: "seatA",
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
			name: "DealIdPresentAndNegativeAdjustedPrice no seat",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  1.0,
					DealID: "dealId",
				},
			},
			givenBidType: VideoInstream,
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|*|dealid": {
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
			name: "DealIdPresentAndNegativeAdjustedPrice with seat",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  1.0,
					DealID: "dealId",
				},
				Seat: "seatA",
			},
			givenBidType: VideoInstream,
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|*|dealid": {
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
			name: "NoDealIdNegativeAdjustedPrice no seat",
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
			name: "NoDealIdNegativeAdjustedPrice with seat",
			givenBidInfo: &adapters.TypedBid{
				Bid: &openrtb2.Bid{
					Price:  1.0,
					DealID: "",
				},
				Seat: "seatA",
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

func TestGetNoSeat(t *testing.T) {
	testCases := []struct {
		name                   string
		givenRuleToAdjustments map[string][]openrtb_ext.Adjustment
		givenBidType           openrtb_ext.BidType
		givenBidderName        openrtb_ext.BidderName
		givenDealId            string
		givenSeat              string
		expected               []openrtb_ext.Adjustment
	}{
		{
			name: "Priority1",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|biddera|dealid": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
				"banner|biddera|*": {
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
				"banner|biddera|*": {
					{
						Type:  AdjustmentTypeStatic,
						Value: 5.0,
					},
				},
				"banner|*|dealid": {
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
				"banner|*|dealid": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|biddera|dealid": {
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
				"*|biddera|dealid": {
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
				"*|biddera|*": {
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
				"*|biddera|*": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|*|dealid": {
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
				"*|*|dealid": {
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
						Type:     AdjustmentTypeCPM,
						Value:    3.0,
						Currency: "USD",
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
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "NoDealID",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|biddera|*": {
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
			givenDealId:     "",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "NoPriorityRulesMatch",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|biddera|dealid": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
				"banner|biddera|*": {
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
			adjArray := get(test.givenRuleToAdjustments, string(test.givenBidType), test.givenSeat, string(test.givenBidderName), test.givenDealId)
			assert.Equal(t, test.expected, adjArray)
		})
	}
}

func TestGetWithSeat(t *testing.T) {
	testCases := []struct {
		name                   string
		givenRuleToAdjustments map[string][]openrtb_ext.Adjustment
		givenBidType           openrtb_ext.BidType
		givenBidderName        openrtb_ext.BidderName
		givenDealId            string
		givenSeat              string
		expected               []openrtb_ext.Adjustment
	}{
		{
			name: "Priority1",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|seata|dealid": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
				"banner|biddera|dealid": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 2.0,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeMultiplier, Value: 1.1}},
		},
		{
			name: "Priority2",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|biddera|dealid": {
					{
						Type:  AdjustmentTypeStatic,
						Value: 5.0,
					},
				},
				"banner|seata|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 2.0,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeStatic, Value: 5.0}},
		},
		{
			name: "Priority3",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|seata|*": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"banner|biddera|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority4",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|biddera|*": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"banner|*|dealid": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority5",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|*|dealid": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|seat|dealid": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority6",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|seata|dealid": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|biddera|dealid": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "dealId",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority7",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|biddera|dealid": {
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
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority8",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|*|*": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|seata|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderB",
			givenDealId:     "dealId",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority9",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|seata|*": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|bidderb|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderB",
			givenDealId:     "dealId",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority10",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|bidderb|*": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|*|dealid": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderB",
			givenDealId:     "dealId",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority11",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|*|dealid": {
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
			givenBidderName: "bidderB",
			givenDealId:     "dealId",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "Priority12",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|*|*": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|biddera|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderB",
			givenDealId:     "dealId",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "NoDealID Priority 1",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|seata|*": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"banner|biddera|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "NoDealID Priority 2",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|biddera|*": {
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
			givenDealId:     "",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "NoDealID Priority 3",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|*|*": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|seata|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "NoDealID Priority 4",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|seata|*": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"*|biddera|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "NoDealID Priority 5",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|biddera|*": {
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
			givenDealId:     "",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "NoDealID Priority 6",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"*|*|*": {
					{
						Type:     AdjustmentTypeCPM,
						Value:    3.0,
						Currency: "USD",
					},
				},
				"banner|bidderb|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeBanner,
			givenBidderName: "bidderA",
			givenDealId:     "",
			givenSeat:       "seatA",
			expected:        []openrtb_ext.Adjustment{{Type: AdjustmentTypeCPM, Value: 3.0, Currency: "USD"}},
		},
		{
			name: "NoPriorityRulesMatch",
			givenRuleToAdjustments: map[string][]openrtb_ext.Adjustment{
				"banner|seata|dealid": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 1.1,
					},
				},
				"banner|seat2|*": {
					{
						Type:  AdjustmentTypeMultiplier,
						Value: 2.0,
					},
				},
			},
			givenBidType:    openrtb_ext.BidTypeVideo,
			givenBidderName: "bidderB",
			givenDealId:     "diffDealId",
			givenSeat:       "seatA",
			expected:        nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			adjArray := get(test.givenRuleToAdjustments, string(test.givenBidType), test.givenSeat, string(test.givenBidderName), test.givenDealId)
			assert.Equal(t, test.expected, adjArray)
		})
	}
}
