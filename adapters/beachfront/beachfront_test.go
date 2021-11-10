package beachfront

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBeachfront, config.Adapter{
		Endpoint:         `https://qa.beachrtb.com/prebid_display`,
		ExtraAdapterInfo: `{"video_endpoint":"https://qa.beachrtb.com/bid.json?exchange_id"}`,
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "beachfronttest", bidder)
}

func TestExtraInfoDefaultWhenEmpty(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBeachfront, config.Adapter{
		Endpoint:         `https://qa.beachrtb.com/prebid_display`,
		ExtraAdapterInfo: ``,
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderBeachfront, _ := bidder.(*BeachfrontAdapter)

	assert.Equal(t, bidderBeachfront.extraInfo.VideoEndpoint, defaultVideoEndpoint)
}

func TestExtraInfoDefaultWhenNotSpecified(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBeachfront, config.Adapter{
		Endpoint:         `https://qa.beachrtb.com/prebid_display`,
		ExtraAdapterInfo: `{"video_endpoint":""}`,
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderBeachfront, _ := bidder.(*BeachfrontAdapter)

	assert.Equal(t, bidderBeachfront.extraInfo.VideoEndpoint, defaultVideoEndpoint)
}

func TestExtraInfoMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderBeachfront, config.Adapter{
		Endpoint:         `https://qa.beachrtb.com/prebid_display`,
		ExtraAdapterInfo: `malformed`,
	})

	assert.Error(t, buildErr)
}

func TestBFBannerRequestWithDifferentBidFloorAttributes(t *testing.T) {
	/*

		testScenarios := []struct {
			bidFloor         float64
			bidFloorCur      string
			extBidFloor			 float64
			setMock          func(m *mock.Mock)
			expectedBidFloor float64
			expectedBidCur   string
			expectedErrors   []error
		}{
			{
				bidFloor:         1,
				bidFloorCur:      "WRONG",
				extBidFloor: 0,
				setMock:          func(m *mock.Mock) { m.On("GetRate", "WRONG", "USD").Return(2.5, errors.New("some error")) },
				expectedBidFloor: 0,
				expectedBidCur:   "",
				expectedErrors: []error{
					&errortypes.BadInput{Message: "Unable to convert provided bid floor currency from WRONG to USD"},
				},
			},
			{
				bidFloor:         1,
				bidFloorCur:      "USD",
				extBidFloor: 0,
				setMock:          func(m *mock.Mock) {},
				expectedBidFloor: 1,
				expectedBidCur:   "USD",
				expectedErrors:   nil,
			},
			{
				bidFloor:         1,
				bidFloorCur:      "EUR",
				extBidFloor: 0,
				setMock:          func(m *mock.Mock) { m.On("GetRate", "EUR", "USD").Return(1.2, nil) },
				expectedBidFloor: 1.2,
				expectedBidCur:   "USD",
				expectedErrors:   nil,
			},
			{
				bidFloor:         0,
				bidFloorCur:      "",
				extBidFloor: 0,
				setMock:          func(m *mock.Mock) {},
				expectedBidFloor: 0,
				expectedBidCur:   "",
				expectedErrors:   nil,
			},
			{
				bidFloor:         -1,
				bidFloorCur:      "CZK",
				extBidFloor: 0,
				setMock:          func(m *mock.Mock) {},
				expectedBidFloor: -1,
				expectedBidCur:   "CZK",
				expectedErrors:   nil,
			},
		}
	*/

	testScenarios := []struct {
		bidFloor         float64
		bidFloorCur      string
		extBidFloor      float64
		setMock          func(m *mock.Mock)
		expectedBidFloor float64
		expectedBidCur   string
		expectedErrors   []error
	}{
		{
			bidFloor:         0.01,
			bidFloorCur:      "USD",
			extBidFloor:      0,
			setMock:          func(m *mock.Mock) {},
			expectedBidFloor: 0,
			expectedBidCur:   "USD",
			expectedErrors:   nil,
		},
	}

	for _, scenario := range testScenarios {
		mockConversions := &mockCurrencyConversion{}
		scenario.setMock(&mockConversions.Mock)

		extraRequestInfo := adapters.ExtraRequestInfo{
			CurrencyConversions: mockConversions,
		}

		bidder := new(BeachfrontAdapter)

		request := &openrtb2.BidRequest{
			ID: "test-request-id",
			Imp: []openrtb2.Imp{{
				ID:          "test-imp-id",
				BidFloorCur: scenario.bidFloorCur,
				BidFloor:    scenario.bidFloor,
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{{W: 300, H: 250}},
				},
				Ext: json.RawMessage(`{"bidder": {
										"appId": "banner-267b23c-96c61b67",
										"bidfloor": ` + fmt.Sprintf("%f", scenario.extBidFloor) + `
                                      }}`),
			}},
			App: &openrtb2.App{
				ID:   "com.test",
				Name: "testApp",
			},
		}

		reqs, errs := bidder.MakeRequests(request, &extraRequestInfo)

		mockConversions.AssertExpectations(t)

		if scenario.expectedErrors == nil {
			bfmReq := &beachfrontBannerRequest{}
			if err := json.Unmarshal(reqs[0].Body, bfmReq); err != nil {
				t.Fatalf("Unexpected error while decoding request: %s", err)
			}
			assert.Equal(t, scenario.expectedBidFloor, bfmReq.Slots[0].Bidfloor)
		} else {
			assert.Equal(t, scenario.expectedErrors, errs)
		}
	}
}

func TestOpenRTBRequestWithDifferentBidFloorAttributes(t *testing.T) {
	/*

		testScenarios := []struct {
			bidFloor         float64
			bidFloorCur      string
			extBidFloor			 float64
			setMock          func(m *mock.Mock)
			expectedBidFloor float64
			expectedBidCur   string
			expectedErrors   []error
		}{
			{
				bidFloor:         1,
				bidFloorCur:      "WRONG",
				extBidFloor: 0,
				setMock:          func(m *mock.Mock) { m.On("GetRate", "WRONG", "USD").Return(2.5, errors.New("some error")) },
				expectedBidFloor: 0,
				expectedBidCur:   "",
				expectedErrors: []error{
					&errortypes.BadInput{Message: "Unable to convert provided bid floor currency from WRONG to USD"},
				},
			},
			{
				bidFloor:         1,
				bidFloorCur:      "USD",
				extBidFloor: 0,
				setMock:          func(m *mock.Mock) {},
				expectedBidFloor: 1,
				expectedBidCur:   "USD",
				expectedErrors:   nil,
			},
			{
				bidFloor:         1,
				bidFloorCur:      "EUR",
				extBidFloor: 0,
				setMock:          func(m *mock.Mock) { m.On("GetRate", "EUR", "USD").Return(1.2, nil) },
				expectedBidFloor: 1.2,
				expectedBidCur:   "USD",
				expectedErrors:   nil,
			},
			{
				bidFloor:         0,
				bidFloorCur:      "",
				extBidFloor: 0,
				setMock:          func(m *mock.Mock) {},
				expectedBidFloor: 0,
				expectedBidCur:   "",
				expectedErrors:   nil,
			},
			{
				bidFloor:         -1,
				bidFloorCur:      "CZK",
				extBidFloor: 0,
				setMock:          func(m *mock.Mock) {},
				expectedBidFloor: -1,
				expectedBidCur:   "CZK",
				expectedErrors:   nil,
			},
		}
	*/

	testScenarios := []struct {
		bidFloor         float64
		bidFloorCur      string
		extBidFloor      float64
		setMock          func(m *mock.Mock)
		expectedBidFloor float64
		expectedBidCur   string
		expectedErrors   []error
	}{
		{
			bidFloor:         0.01,
			bidFloorCur:      "USD",
			extBidFloor:      0,
			setMock:          func(m *mock.Mock) {},
			expectedBidFloor: 0,
			expectedBidCur:   "USD",
			expectedErrors:   nil,
		},
	}

	for _, scenario := range testScenarios {
		mockConversions := &mockCurrencyConversion{}
		scenario.setMock(&mockConversions.Mock)

		extraRequestInfo := adapters.ExtraRequestInfo{
			CurrencyConversions: mockConversions,
		}

		bidder := new(BeachfrontAdapter)

		request := &openrtb2.BidRequest{
			ID: "test-request-id",
			Imp: []openrtb2.Imp{{
				ID:          "test-imp-id",
				BidFloorCur: scenario.bidFloorCur,
				BidFloor:    scenario.bidFloor,
				Video: &openrtb2.Video{
					W:     300,
					H:     250,
					MIMEs: []string{"video/mp4"},
				},
				Ext: json.RawMessage(`{"bidder": {
										"appId": "banner-267b23c-96c61b67",
										"bidfloor": ` + fmt.Sprintf("%f", scenario.extBidFloor) + `
                                      }}`),
			}},
			App: &openrtb2.App{
				ID:   "com.test",
				Name: "testApp",
			},
		}

		reqs, errs := bidder.MakeRequests(request, &extraRequestInfo)

		mockConversions.AssertExpectations(t)

		if scenario.expectedErrors == nil {
			bfmReq := &openrtb2.BidRequest{}
			if err := json.Unmarshal(reqs[0].Body, bfmReq); err != nil {
				t.Fatalf("Unexpected error while decoding request: %s", err)
			}
			assert.Equal(t, scenario.expectedBidFloor, bfmReq.Imp[0].BidFloor)
			assert.Equal(t, scenario.expectedBidFloor, bfmReq.Imp[0].BidFloor)
		} else {
			assert.Equal(t, scenario.expectedErrors, errs)
		}
	}
}

type mockCurrencyConversion struct {
	mock.Mock
}

func (m mockCurrencyConversion) GetRate(from string, to string) (float64, error) {
	args := m.Called(from, to)
	return args.Get(0).(float64), args.Error(1)
}

func (m mockCurrencyConversion) GetRates() *map[string]map[string]float64 {
	args := m.Called()
	return args.Get(0).(*map[string]map[string]float64)
}
