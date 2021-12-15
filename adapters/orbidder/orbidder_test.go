package orbidder

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/stretchr/testify/mock"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
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
			if tc.expectErr {
				assert.Error(t, err)
			}
			if !tc.expectErr {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderOrbidder, config.Adapter{
		Endpoint: "https://orbidder-test"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "orbiddertest", bidder)
}

var testCasesCurrency = map[string]struct {
	imp         openrtb2.Imp
	setMock     func(m *mock.Mock)
	expectedImp openrtb2.Imp
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
	},
}

var testCasesExtension = map[string]struct {
	imp       openrtb2.Imp
	expectErr bool
}{
	"Valid Orbidder Extension": {
		imp: openrtb2.Imp{
			Ext: json.RawMessage(`{"bidder":{"accountId":"orbidder-test", "placementId":"center-banner", "bidfloor": 0.1}}`),
		},
		expectErr: false,
	},
	"Invalid Orbidder Extension": {
		imp: openrtb2.Imp{
			Ext: json.RawMessage(`{"there's'":{"something":"strange", "in the":"neighbourhood", "who you gonna call?": 0.1}}`),
		},
		expectErr: true,
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
			assert.NoError(t, err)
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
