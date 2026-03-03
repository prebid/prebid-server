package orbidder

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/mock"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalOrbidderExtImp(t *testing.T) {
	ext := json.RawMessage(`{"accountId":"orbidder-test", "placementId":"center-banner", "bidfloor": 0.1}`)
	impExt := new(openrtb_ext.ExtImpOrbidder)

	assert.NoError(t, json.Unmarshal(ext, impExt))
	assert.Equal(t, &openrtb_ext.ExtImpOrbidder{
		AccountId:   "orbidder-test",
		PlacementId: "center-banner",
		BidFloor:    0.1,
	}, impExt)
}

func TestPreprocessExtensions(t *testing.T) {
	for name, tc := range testCasesExtension {
		t.Run(name, func(t *testing.T) {
			imp := tc.imp
			err := preprocessExtensions(&imp)
			tc.assertError(t, err)
		})
	}
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderOrbidder, config.Adapter{
		Endpoint: "https://orbidder-test"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "orbiddertest", bidder)
}

var testCasesCurrency = map[string]struct {
	imp         openrtb2.Imp
	setMock     func(m *mock.Mock)
	expectedImp openrtb2.Imp
	assertError func(t assert.TestingT, err error, msgAndArgs ...interface{}) bool
}{
	"EUR: no bidfloor, no currency": {
		imp: openrtb2.Imp{
			BidFloor:    0,
			BidFloorCur: "",
		},
		setMock: func(m *mock.Mock) {},
		expectedImp: openrtb2.Imp{
			BidFloor:    0,
			BidFloorCur: "EUR",
		},
		assertError: assert.NoError,
	},
	"EUR: bidfloor, no currency": {
		imp: openrtb2.Imp{
			BidFloor:    1,
			BidFloorCur: "",
		},
		setMock: func(m *mock.Mock) {},
		expectedImp: openrtb2.Imp{
			BidFloor:    1,
			BidFloorCur: "EUR",
		},
		assertError: assert.NoError,
	},
	"EUR: bidfloor and currency": {
		imp: openrtb2.Imp{
			BidFloor:    1,
			BidFloorCur: "EUR",
		},
		setMock: func(m *mock.Mock) {},
		expectedImp: openrtb2.Imp{
			BidFloor:    1,
			BidFloorCur: "EUR",
		},
		assertError: assert.NoError,
	},
	"USD: bidfloor with currency": {
		imp: openrtb2.Imp{
			BidFloor:    1,
			BidFloorCur: "USD",
		},
		setMock: func(m *mock.Mock) {
			m.On("GetRate", "USD", "EUR").Return(2.5, nil)
		},
		expectedImp: openrtb2.Imp{
			BidFloor:    2.5,
			BidFloorCur: "EUR",
		},
		assertError: assert.NoError,
	},
	"USD: no bidfloor": {
		imp: openrtb2.Imp{
			BidFloor:    0,
			BidFloorCur: "USD",
		},
		setMock: func(m *mock.Mock) {},
		expectedImp: openrtb2.Imp{
			BidFloor:    0,
			BidFloorCur: "EUR",
		},
		assertError: assert.NoError,
	},
	"ABC: invalid currency code": {
		imp: openrtb2.Imp{
			BidFloor:    1,
			BidFloorCur: "ABC",
		},
		setMock: func(m *mock.Mock) {
			m.On("GetRate", "ABC", "EUR").Return(0.0, errors.New("currency conversion error"))
		},
		expectedImp: openrtb2.Imp{
			BidFloor:    1,
			BidFloorCur: "ABC",
		},
		assertError: assert.Error,
	},
}

var testCasesExtension = map[string]struct {
	imp         openrtb2.Imp
	assertError func(t assert.TestingT, err error, msgAndArgs ...interface{}) bool
}{
	"Valid Orbidder Extension": {
		imp: openrtb2.Imp{
			Ext: json.RawMessage(`{"bidder":{"accountId":"orbidder-test", "placementId":"center-banner", "bidfloor": 0.1}}`),
		},
		assertError: assert.NoError,
	},
	"Invalid Orbidder Extension": {
		imp: openrtb2.Imp{
			Ext: json.RawMessage(`{"there's'":{"something":"strange", "in the":"neighbourhood", "who you gonna call?": 0.1}}`),
		},
		assertError: assert.Error,
	},
}

func TestPreprocessBidFloorCurrency(t *testing.T) {
	for name, tc := range testCasesCurrency {
		t.Run(name, func(t *testing.T) {
			imp := tc.imp
			mockConversions := &mockCurrencyConversion{}
			tc.setMock(&mockConversions.Mock)
			extraRequestInfo := adapters.ExtraRequestInfo{
				CurrencyConversions: mockConversions,
			}
			err := preprocessBidFloorCurrency(&imp, &extraRequestInfo)
			assert.True(t, mockConversions.AssertExpectations(t))
			tc.assertError(t, err)
			assert.Equal(t, tc.expectedImp, imp)
		})
	}
}

type mockCurrencyConversion struct {
	mock.Mock
}

func (m *mockCurrencyConversion) GetRate(from string, to string) (float64, error) {
	args := m.Called(from, to)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockCurrencyConversion) GetRates() *map[string]map[string]float64 {
	args := m.Called()
	return args.Get(0).(*map[string]map[string]float64)
}
