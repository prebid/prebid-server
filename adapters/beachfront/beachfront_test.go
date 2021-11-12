package beachfront

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
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

func TestRequestWithDifferentBidFloorAttributes(t *testing.T) {
	scenarios := []struct {
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
			expectedBidCur:   "",
			expectedErrors:   nil,
		},
		{
			bidFloor:         0,
			bidFloorCur:      "USD",
			extBidFloor:      0.01,
			setMock:          func(m *mock.Mock) {},
			expectedBidFloor: 0,
			expectedBidCur:   "",
			expectedErrors:   nil,
		},
		{
			bidFloor:         0,
			bidFloorCur:      "USD",
			extBidFloor:      0,
			setMock:          func(m *mock.Mock) {},
			expectedBidFloor: 0,
			expectedBidCur:   "",
			expectedErrors:   nil,
		},
		{
			bidFloor:         2.01,
			bidFloorCur:      "USD",
			extBidFloor:      1.90,
			setMock:          func(m *mock.Mock) {},
			expectedBidFloor: 2.01,
			expectedBidCur:   "USD",
			expectedErrors:   nil,
		},
		{
			bidFloor:         1.01,
			bidFloorCur:      "USD",
			extBidFloor:      1.90,
			setMock:          func(m *mock.Mock) {},
			expectedBidFloor: 1.90,
			expectedBidCur:   "USD",
			expectedErrors:   nil,
		},
		{
			bidFloor:         1.00,
			bidFloorCur:      "XYZ",
			extBidFloor:      0.99,
			setMock:          func(m *mock.Mock) { m.On("GetRate", "XYZ", "USD").Return(2.5, errors.New("some error")) },
			expectedBidFloor: 0.99,
			expectedBidCur:   "USD",
			expectedErrors: []error{
				&errortypes.Warning{Message: "The following error was recieved from the currency converter while attempting to convert the imp.bidfloor value of 1.00 from XYZ to USD: \nsome error\n The provided value of imp.ext.beachfront.bidfloor, 0.99 USD is being used as a fallback."},
			},
		},
		{
			bidFloor:         1.00,
			bidFloorCur:      "XYZ",
			extBidFloor:      0,
			setMock:          func(m *mock.Mock) { m.On("GetRate", "XYZ", "USD").Return(2.5, errors.New("some error")) },
			expectedBidFloor: 0,
			expectedBidCur:   "USD",
			expectedErrors: []error{
				&errortypes.BadInput{Message: "The following error was recieved from the currency converter while attempting to convert the imp.bidfloor value of 1.00 from XYZ to USD: \nsome error\n A value of imp.ext.beachfront.bidfloor was not provided. The bid is being skipped."},
			},
		},
		{
			bidFloor:         1.00,
			bidFloorCur:      "XYZ",
			extBidFloor:      1.99,
			setMock:          func(m *mock.Mock) { m.On("GetRate", "XYZ", "USD").Return(1.5, errors.New("some error")) },
			expectedBidFloor: 1.99,
			expectedBidCur:   "USD",
			expectedErrors: []error{
				&errortypes.Warning{Message: "The following error was recieved from the currency converter while attempting to convert the imp.bidfloor value of 1.00 from XYZ to USD: \nsome error\n The provided value of imp.ext.beachfront.bidfloor, 1.99 USD is being used as a fallback."},
			},
		},
		{
			bidFloor:         1.03,
			bidFloorCur:      "EUR",
			extBidFloor:      1.98,
			setMock:          func(m *mock.Mock) { m.On("GetRate", "EUR", "USD").Return(1.2, nil) },
			expectedBidFloor: 1.98,
			expectedBidCur:   "USD",
			expectedErrors:   nil,
		},
		{
			bidFloor:         1.80,
			bidFloorCur:      "EUR",
			extBidFloor:      1.98,
			setMock:          func(m *mock.Mock) { m.On("GetRate", "EUR", "USD").Return(1.2, nil) },
			expectedBidFloor: 2.16,
			expectedBidCur:   "USD",
			expectedErrors:   nil,
		},
	}

	for _, scenario := range scenarios {
		mockConversions := &mockCurrencyConversion{}
		scenario.setMock(&mockConversions.Mock)

		extraRequestInfo := adapters.ExtraRequestInfo{
			CurrencyConversions: mockConversions,
		}

		bidder := new(BeachfrontAdapter)

		videoRequest := &openrtb2.BidRequest{
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
										"appId": "video-2cf761-83b5a0",
										"bidfloor": ` + fmt.Sprintf("%.2f", scenario.extBidFloor) + `
                                      }}`),
			}},
			App: &openrtb2.App{
				ID:   "com.test",
				Name: "testApp",
			},
		}

		bannerRequest := &openrtb2.BidRequest{
			ID: "test-request-id",
			Imp: []openrtb2.Imp{{
				ID:          "test-imp-id",
				BidFloorCur: scenario.bidFloorCur,
				BidFloor:    scenario.bidFloor,
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{{W: 300, H: 250}},
				},
				Ext: json.RawMessage(`{"bidder": {
										"appId": "banner-27b23c-96cb67",
										"bidfloor": ` + fmt.Sprintf("%.2f", scenario.extBidFloor) + `
                                      }}`),
			}},
			App: &openrtb2.App{
				ID:   "com.test",
				Name: "testApp",
			},
		}

		reqs, errs := bidder.MakeRequests(bannerRequest, &extraRequestInfo)

		mockConversions.AssertExpectations(t)

		if scenario.expectedErrors == nil {
			bfmBannerReq := &beachfrontBannerRequest{}
			if err := json.Unmarshal(reqs[0].Body, bfmBannerReq); err != nil {
				t.Fatalf("Unexpected error while decoding request: %s", err)
			}
			assert.Equal(t, scenario.expectedBidFloor, bfmBannerReq.Slots[0].Bidfloor)
		} else {
			assert.Equal(t, scenario.expectedErrors, errs)
		}

		reqs, errs = bidder.MakeRequests(videoRequest, &extraRequestInfo)

		if scenario.expectedErrors == nil {
			bfmVideoReq := &openrtb2.BidRequest{}
			if err := json.Unmarshal(reqs[0].Body, bfmVideoReq); err != nil {
				t.Fatalf("Unexpected error while decoding request: %s", err)
			}
			assert.Equal(t, scenario.expectedBidFloor, bfmVideoReq.Imp[0].BidFloor)
			assert.Equal(t, scenario.expectedBidCur, bfmVideoReq.Imp[0].BidFloorCur)
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
