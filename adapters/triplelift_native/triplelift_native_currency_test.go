package triplelift_native

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

func TestResolveBidFloorCurrency(t *testing.T) {
	cases := map[string]struct {
		imp         openrtb2.Imp
		setMock     func(m *mock.Mock)
		expectedImp openrtb2.Imp
		assertError assert.ErrorAssertionFunc
	}{
		"no floor": {
			imp:         openrtb2.Imp{BidFloor: 0, BidFloorCur: "EUR"},
			setMock:     func(m *mock.Mock) {},
			expectedImp: openrtb2.Imp{BidFloor: 0, BidFloorCur: "EUR"},
			assertError: assert.NoError,
		},
		"USD floor passthrough": {
			imp:         openrtb2.Imp{BidFloor: 3.0, BidFloorCur: "USD"},
			setMock:     func(m *mock.Mock) {},
			expectedImp: openrtb2.Imp{BidFloor: 3.0, BidFloorCur: "USD"},
			assertError: assert.NoError,
		},
		"EUR converts to USD": {
			imp: openrtb2.Imp{BidFloor: 2.0, BidFloorCur: "EUR"},
			setMock: func(m *mock.Mock) {
				m.On("GetRate", "EUR", "USD").Return(1.08, nil)
			},
			expectedImp: openrtb2.Imp{BidFloor: 2.16, BidFloorCur: "USD"},
			assertError: assert.NoError,
		},
		"conversion error": {
			imp: openrtb2.Imp{BidFloor: 1.0, BidFloorCur: "XYZ"},
			setMock: func(m *mock.Mock) {
				m.On("GetRate", "XYZ", "USD").Return(0.0, errors.New("rate not found"))
			},
			expectedImp: openrtb2.Imp{BidFloor: 1.0, BidFloorCur: "XYZ"},
			assertError: assert.Error,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			imp := tc.imp
			m := &mockCurrencyConversion{}
			tc.setMock(&m.Mock)
			reqInfo := adapters.ExtraRequestInfo{CurrencyConversions: m}
			err := resolveBidFloorCurrency(&imp, &reqInfo)
			tc.assertError(t, err)
			assert.Equal(t, tc.expectedImp.BidFloor, imp.BidFloor)
			assert.Equal(t, tc.expectedImp.BidFloorCur, imp.BidFloorCur)
			m.AssertExpectations(t)
		})
	}
}
