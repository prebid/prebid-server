package triplelift

import (
	"errors"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCurrencyConversion struct {
	mock.Mock
}

func (m *mockCurrencyConversion) GetRate(from, to string) (float64, error) {
	args := m.Called(from, to)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockCurrencyConversion) GetRates() *map[string]map[string]float64 {
	args := m.Called()
	return args.Get(0).(*map[string]map[string]float64)
}

var currencyTestCases = map[string]struct {
	imp         openrtb2.Imp
	setMock     func(m *mock.Mock)
	expectedImp openrtb2.Imp
	assertError assert.ErrorAssertionFunc
}{
	"no floor, no currency": {
		imp:         openrtb2.Imp{BidFloor: 0, BidFloorCur: ""},
		setMock:     func(m *mock.Mock) {},
		expectedImp: openrtb2.Imp{BidFloor: 0, BidFloorCur: ""},
		assertError: assert.NoError,
	},
	"floor, empty currency (implicit USD)": {
		imp:         openrtb2.Imp{BidFloor: 1.25, BidFloorCur: ""},
		setMock:     func(m *mock.Mock) {},
		expectedImp: openrtb2.Imp{BidFloor: 1.25, BidFloorCur: "USD"},
		assertError: assert.NoError,
	},
	"USD floor, no conversion needed": {
		imp:         openrtb2.Imp{BidFloor: 2.0, BidFloorCur: "USD"},
		setMock:     func(m *mock.Mock) {},
		expectedImp: openrtb2.Imp{BidFloor: 2.0, BidFloorCur: "USD"},
		assertError: assert.NoError,
	},
	"usd lowercase - no conversion needed": {
		imp:         openrtb2.Imp{BidFloor: 2.0, BidFloorCur: "usd"},
		setMock:     func(m *mock.Mock) {},
		expectedImp: openrtb2.Imp{BidFloor: 2.0, BidFloorCur: "USD"},
		assertError: assert.NoError,
	},
	"EUR floor converts to USD": {
		imp: openrtb2.Imp{BidFloor: 1.0, BidFloorCur: "EUR"},
		setMock: func(m *mock.Mock) {
			m.On("GetRate", "EUR", "USD").Return(1.1, nil)
		},
		expectedImp: openrtb2.Imp{BidFloor: 1.1, BidFloorCur: "USD"},
		assertError: assert.NoError,
	},
	"unknown currency returns error": {
		imp: openrtb2.Imp{BidFloor: 1.0, BidFloorCur: "ABC"},
		setMock: func(m *mock.Mock) {
			m.On("GetRate", "ABC", "USD").Return(0.0, errors.New("conversion rate not found"))
		},
		expectedImp: openrtb2.Imp{BidFloor: 1.0, BidFloorCur: "ABC"},
		assertError: assert.Error,
	},
}

func TestResolveBidFloorCurrency(t *testing.T) {
	for name, tc := range currencyTestCases {
		t.Run(name, func(t *testing.T) {
			imp := tc.imp
			mockConversions := &mockCurrencyConversion{}
			tc.setMock(&mockConversions.Mock)

			reqInfo := adapters.ExtraRequestInfo{CurrencyConversions: mockConversions}
			err := resolveBidFloorCurrency(&imp, &reqInfo)

			tc.assertError(t, err)
			assert.Equal(t, tc.expectedImp.BidFloor, imp.BidFloor)
			assert.Equal(t, tc.expectedImp.BidFloorCur, imp.BidFloorCur)
			mockConversions.AssertExpectations(t)
		})
	}
}
