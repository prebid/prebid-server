package rubicon

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"testing"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type rubiAppendTrackerUrlTestScenario struct {
	source   string
	tracker  string
	expected string
}

type rubiPopulateFpdAttributesScenario struct {
	source json.RawMessage
	target map[string]interface{}
	result map[string]interface{}
}

type rubiSetNetworkIdTestScenario struct {
	bidExt            *openrtb_ext.ExtBidPrebid
	buyer             string
	expectedNetworkId int64
	isNetworkIdSet    bool
}

type rubiBidInfo struct {
	domain        string
	page          string
	deviceIP      string
	deviceUA      string
	buyerUID      string
	devicePxRatio float64
}

var rubidata rubiBidInfo

func getTestSizes() map[int]openrtb2.Format {
	return map[int]openrtb2.Format{
		15: {W: 300, H: 250},
		10: {W: 300, H: 600},
		2:  {W: 728, H: 91},
		9:  {W: 160, H: 600},
		8:  {W: 120, H: 600},
		33: {W: 180, H: 500},
		43: {W: 320, H: 50},
	}
}

func TestParseSizes(t *testing.T) {
	SIZE_ID := getTestSizes()

	sizes := []openrtb2.Format{
		SIZE_ID[10],
		SIZE_ID[15],
	}
	primary, alt, err := parseRubiconSizes(sizes)
	assert.Nil(t, err, "Parsing error: %v", err)
	assert.Equal(t, 15, primary, "Primary %d != 15", primary)
	assert.Equal(t, 1, len(alt), "Alt not len 1")
	assert.Equal(t, 10, alt[0], "Alt not 10: %d", alt[0])

	sizes = []openrtb2.Format{
		{
			W: 1111,
			H: 2222,
		},
		SIZE_ID[15],
	}
	primary, alt, err = parseRubiconSizes(sizes)
	assert.Nil(t, err, "Shouldn't have thrown error for invalid size 1111x1111 since we still have a valid one")
	assert.Equal(t, 15, primary, "Primary %d != 15", primary)
	assert.Equal(t, 0, len(alt), "Alt len %d != 0", len(alt))

	sizes = []openrtb2.Format{
		SIZE_ID[15],
	}
	primary, alt, err = parseRubiconSizes(sizes)
	assert.Nil(t, err, "Parsing error: %v", err)
	assert.Equal(t, 15, primary, "Primary %d != 15", primary)
	assert.Equal(t, 0, len(alt), "Alt len %d != 0", len(alt))

	sizes = []openrtb2.Format{
		{
			W: 1111,
			H: 1222,
		},
	}
	primary, alt, err = parseRubiconSizes(sizes)
	assert.NotNil(t, err, "Parsing error: %v", err)
	assert.Equal(t, 0, primary, "Primary %d != 15", primary)
	assert.Equal(t, 0, len(alt), "Alt len %d != 0", len(alt))
}

func TestMASAlgorithm(t *testing.T) {
	SIZE_ID := getTestSizes()
	type output struct {
		primary int
		alt     []int
		ok      bool
	}
	type testStub struct {
		input  []openrtb2.Format
		output output
	}

	testStubs := []testStub{
		{
			[]openrtb2.Format{
				SIZE_ID[2],
				SIZE_ID[9],
			},
			output{2, []int{9}, false},
		},
		{
			[]openrtb2.Format{

				SIZE_ID[9],
				SIZE_ID[15],
			},
			output{15, []int{9}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[2],
				SIZE_ID[15],
			},
			output{15, []int{2}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[15],
				SIZE_ID[9],
				SIZE_ID[2],
			},
			output{15, []int{2, 9}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[10],
				SIZE_ID[9],
				SIZE_ID[2],
			},
			output{2, []int{10, 9}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[33],
				SIZE_ID[8],
				SIZE_ID[15],
			},
			output{15, []int{33, 8}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[33],
				SIZE_ID[8],
				SIZE_ID[9],
				SIZE_ID[2],
			},
			output{2, []int{33, 8, 9}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[33],
				SIZE_ID[8],
				SIZE_ID[9],
			},
			output{9, []int{33, 8}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[33],
				SIZE_ID[8],
				SIZE_ID[2],
			},
			output{2, []int{33, 8}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[33],
				SIZE_ID[2],
			},
			output{2, []int{33}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[8],
			},
			output{8, []int{}, false},
		},
		{
			[]openrtb2.Format{},
			output{0, []int{}, true},
		},
		{
			[]openrtb2.Format{
				{W: 1111,
					H: 2345,
				},
			},
			output{0, []int{}, true},
		},
	}

	for _, test := range testStubs {
		prim, alt, err := parseRubiconSizes(test.input)

		assert.Equal(t, test.output.primary, prim,
			"Error in parsing rubicon sizes: MAS algorithm fail at primary: testcase %v", test.input)

		assert.Equal(t, len(test.output.alt), len(alt),
			"Error in parsing rubicon sizes: MAS Algorithm fail at alt: testcase %v", test.input)

		assert.False(t, err != nil && !test.output.ok,
			"Error in parsing rubicon sizes: MAS Algorithm fail at throwing error: testcase %v", test.input)

		assert.False(t, err == nil && test.output.ok,
			"Error in parsing rubicon sizes: MAS Algorithm fail at throwing error: testcase %v", test.input)
	}
}

func TestAppendTracker(t *testing.T) {
	testScenarios := []rubiAppendTrackerUrlTestScenario{
		{
			source:   "http://test.url/",
			tracker:  "prebid",
			expected: "http://test.url/?tk_xint=prebid",
		},
		{
			source:   "http://test.url/?hello=true",
			tracker:  "prebid",
			expected: "http://test.url/?hello=true&tk_xint=prebid",
		},
	}

	for _, scenario := range testScenarios {
		res := appendTrackerToUrl(scenario.source, scenario.tracker)
		assert.Equal(t, scenario.expected, res, "Failed to convert '%s' to '%s'", res, scenario.expected)
	}
}

func TestResolveVideoSizeId(t *testing.T) {
	testScenarios := []struct {
		placement   openrtb2.VideoPlacementType
		instl       int8
		impId       string
		expected    int
		expectedErr error
	}{
		{
			placement:   1,
			instl:       1,
			impId:       "impId",
			expected:    201,
			expectedErr: nil,
		},
		{
			placement:   3,
			instl:       1,
			impId:       "impId",
			expected:    203,
			expectedErr: nil,
		},
		{
			placement:   4,
			instl:       1,
			impId:       "impId",
			expected:    202,
			expectedErr: nil,
		},
		{
			placement: 4,
			instl:     3,
			impId:     "impId",
			expectedErr: &errortypes.BadInput{
				Message: "video.size_id can not be resolved in impression with id : impId",
			},
		},
	}

	for _, scenario := range testScenarios {
		res, err := resolveVideoSizeId(scenario.placement, scenario.instl, scenario.impId)
		assert.Equal(t, scenario.expected, res)
		assert.Equal(t, scenario.expectedErr, err)
	}
}

func TestOpenRTBRequestWithDifferentBidFloorAttributes(t *testing.T) {
	testScenarios := []struct {
		bidFloor         float64
		bidFloorCur      string
		setMock          func(m *mock.Mock)
		expectedBidFloor float64
		expectedBidCur   string
		expectedErrors   []error
	}{
		{
			bidFloor:         1,
			bidFloorCur:      "WRONG",
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
			setMock:          func(m *mock.Mock) {},
			expectedBidFloor: 1,
			expectedBidCur:   "USD",
			expectedErrors:   nil,
		},
		{
			bidFloor:         1,
			bidFloorCur:      "EUR",
			setMock:          func(m *mock.Mock) { m.On("GetRate", "EUR", "USD").Return(1.2, nil) },
			expectedBidFloor: 1.2,
			expectedBidCur:   "USD",
			expectedErrors:   nil,
		},
		{
			bidFloor:         0,
			bidFloorCur:      "",
			setMock:          func(m *mock.Mock) {},
			expectedBidFloor: 0,
			expectedBidCur:   "",
			expectedErrors:   nil,
		},
		{
			bidFloor:         -1,
			bidFloorCur:      "CZK",
			setMock:          func(m *mock.Mock) {},
			expectedBidFloor: -1,
			expectedBidCur:   "CZK",
			expectedErrors:   nil,
		},
	}

	for _, scenario := range testScenarios {
		mockConversions := &mockCurrencyConversion{}
		scenario.setMock(&mockConversions.Mock)

		extraRequestInfo := adapters.ExtraRequestInfo{
			CurrencyConversions: mockConversions,
		}

		SIZE_ID := getTestSizes()
		bidder := new(RubiconAdapter)

		request := &openrtb2.BidRequest{
			ID: "test-request-id",
			Imp: []openrtb2.Imp{{
				ID:          "test-imp-id",
				BidFloorCur: scenario.bidFloorCur,
				BidFloor:    scenario.bidFloor,
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						SIZE_ID[15],
						SIZE_ID[10],
					},
				},
				Ext: json.RawMessage(`{"bidder": {
										"zoneId": 8394,
										"siteId": 283282,
										"accountId": 7891
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
			rubiconReq := &openrtb2.BidRequest{}
			if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
				t.Fatalf("Unexpected error while decoding request: %s", err)
			}
			assert.Equal(t, scenario.expectedBidFloor, rubiconReq.Imp[0].BidFloor)
			assert.Equal(t, scenario.expectedBidCur, rubiconReq.Imp[0].BidFloorCur)
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

func TestOpenRTBRequest(t *testing.T) {
	SIZE_ID := getTestSizes()
	bidder := new(RubiconAdapter)

	rubidata = rubiBidInfo{
		domain:        "nytimes.com",
		page:          "https://www.nytimes.com/2017/05/04/movies/guardians-of-the-galaxy-2-review-chris-pratt.html?hpw&rref=movies&action=click&pgtype=Homepage&module=well-region&region=bottom-well&WT.nav=bottom-well&_r=0",
		deviceIP:      "25.91.96.36",
		deviceUA:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_4) AppleWebKit/603.1.30 (KHTML, like Gecko) Version/10.1 Safari/603.1.30",
		buyerUID:      "need-an-actual-rp-id",
		devicePxRatio: 4.0,
	}

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-banner-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					SIZE_ID[15],
					SIZE_ID[10],
				},
			},
			Ext: json.RawMessage(`{"bidder": {
				"zoneId": 8394,
				"siteId": 283282,
				"accountId": 7891,
				"inventory": {"key1" : "val1"},
				"visitor": {"key2" : "val2"}
			}}`),
		}, {
			ID: "test-imp-video-id",
			Video: &openrtb2.Video{
				W:           640,
				H:           360,
				MIMEs:       []string{"video/mp4"},
				MinDuration: 15,
				MaxDuration: 30,
			},
			Ext: json.RawMessage(`{"bidder": {
				"zoneId": 7780,
				"siteId": 283282,
				"accountId": 7891,
				"inventory": {"key1" : "val1"},
				"visitor": {"key2" : "val2"},
				"video": {
					"language": "en",
					"playerHeight": 360,
					"playerWidth": 640,
					"size_id": 203,
					"skip": 1,
					"skipdelay": 5
				}
			}}`),
		}},
		App: &openrtb2.App{
			ID:   "com.test",
			Name: "testApp",
		},
		Device: &openrtb2.Device{
			PxRatio: rubidata.devicePxRatio,
		},
		User: &openrtb2.User{
			Ext: json.RawMessage(`{
				"eids": [{
                    "source": "pubcid",
                    "id": "2402fc76-7b39-4f0e-bfc2-060ef7693648"
				}]
            }`),
		},
		Ext: json.RawMessage(`{"prebid": {}}`),
	}

	reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs, "Got unexpected errors while building HTTP requests: %v", errs)
	assert.Equal(t, 2, len(reqs), "Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 2)

	for i := 0; i < len(reqs); i++ {
		httpReq := reqs[i]
		assert.Equal(t, "POST", httpReq.Method, "Expected a POST message. Got %s", httpReq.Method)

		var rpRequest openrtb2.BidRequest
		if err := json.Unmarshal(httpReq.Body, &rpRequest); err != nil {
			t.Fatalf("Failed to unmarshal HTTP request: %v", rpRequest)
		}

		assert.Equal(t, request.ID, rpRequest.ID, "Bad Request ID. Expected %s, Got %s", request.ID, rpRequest.ID)
		assert.Equal(t, 1, len(rpRequest.Imp), "Wrong len(request.Imp). Expected %d, Got %d", len(request.Imp), len(rpRequest.Imp))
		assert.Nil(t, rpRequest.Cur, "Wrong request.Cur. Expected nil, Got %s", rpRequest.Cur)
		assert.Nil(t, rpRequest.Ext, "Wrong request.ext. Expected nil, Got %v", rpRequest.Ext)

		if rpRequest.Imp[0].ID == "test-imp-banner-id" {
			var rpExt rubiconBannerExt
			if err := json.Unmarshal(rpRequest.Imp[0].Ext, &rpExt); err != nil {
				t.Fatal("Error unmarshalling request from the outgoing request.")
			}

			assert.Equal(t, int64(300), rpRequest.Imp[0].Banner.Format[0].W,
				"Banner width does not match. Expected %d, Got %d", 300, rpRequest.Imp[0].Banner.Format[0].W)

			assert.Equal(t, int64(250), rpRequest.Imp[0].Banner.Format[0].H,
				"Banner height does not match. Expected %d, Got %d", 250, rpRequest.Imp[0].Banner.Format[0].H)

			assert.Equal(t, int64(300), rpRequest.Imp[0].Banner.Format[1].W,
				"Banner width does not match. Expected %d, Got %d", 300, rpRequest.Imp[0].Banner.Format[1].W)

			assert.Equal(t, int64(600), rpRequest.Imp[0].Banner.Format[1].H,
				"Banner height does not match. Expected %d, Got %d", 600, rpRequest.Imp[0].Banner.Format[1].H)
		} else if rpRequest.Imp[0].ID == "test-imp-video-id" {
			var rpExt rubiconVideoExt
			if err := json.Unmarshal(rpRequest.Imp[0].Ext, &rpExt); err != nil {
				t.Fatal("Error unmarshalling request from the outgoing request.")
			}

			assert.Equal(t, int64(640), rpRequest.Imp[0].Video.W,
				"Video width does not match. Expected %d, Got %d", 640, rpRequest.Imp[0].Video.W)

			assert.Equal(t, int64(360), rpRequest.Imp[0].Video.H,
				"Video height does not match. Expected %d, Got %d", 360, rpRequest.Imp[0].Video.H)

			assert.Equal(t, "video/mp4", rpRequest.Imp[0].Video.MIMEs[0], "Video MIMEs do not match. Expected %s, Got %s", "video/mp4", rpRequest.Imp[0].Video.MIMEs[0])

			assert.Equal(t, int64(15), rpRequest.Imp[0].Video.MinDuration,
				"Video min duration does not match. Expected %d, Got %d", 15, rpRequest.Imp[0].Video.MinDuration)

			assert.Equal(t, int64(30), rpRequest.Imp[0].Video.MaxDuration,
				"Video max duration does not match. Expected %d, Got %d", 30, rpRequest.Imp[0].Video.MaxDuration)
		}

		assert.NotNil(t, rpRequest.User.Ext, "User.Ext object should not be nil.")

		var userExt rubiconUserExt
		if err := json.Unmarshal(rpRequest.User.Ext, &userExt); err != nil {
			t.Fatal("Error unmarshalling request.user.ext object.")
		}

		assert.NotNil(t, userExt.Eids)
		assert.Equal(t, 1, len(userExt.Eids), "Eids values are not as expected!")
		assert.Contains(t, userExt.Eids, openrtb_ext.ExtUserEid{Source: "pubcid", ID: "2402fc76-7b39-4f0e-bfc2-060ef7693648"})
	}
}

func TestOpenRTBRequestWithBannerImpEvenIfImpHasVideo(t *testing.T) {
	SIZE_ID := getTestSizes()
	bidder := new(RubiconAdapter)

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					SIZE_ID[15],
					SIZE_ID[10],
				},
			},
			Video: &openrtb2.Video{
				W:     640,
				H:     360,
				MIMEs: []string{"video/mp4"},
			},
			Ext: json.RawMessage(`{"bidder": {
				"zoneId": 8394,
				"siteId": 283282,
				"accountId": 7891,
				"inventory": {"key1" : "val1"},
				"visitor": {"key2" : "val2"}
			}}`),
		}},
		App: &openrtb2.App{
			ID:   "com.test",
			Name: "testApp",
		},
	}

	reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs, "Got unexpected errors while building HTTP requests: %v", errs)

	assert.Equal(t, 1, len(reqs), "Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 1)

	rubiconReq := &openrtb2.BidRequest{}
	if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
		t.Fatalf("Unexpected error while decoding request: %s", err)
	}

	assert.Equal(t, 1, len(rubiconReq.Imp), "Unexpected number of request impressions. Got %d. Expected %d", len(rubiconReq.Imp), 1)

	assert.Nil(t, rubiconReq.Imp[0].Video, "Unexpected video object in request impression")

	assert.NotNil(t, rubiconReq.Imp[0].Banner, "Banner object must be in request impression")
}

func TestOpenRTBRequestWithImpAndAdSlotIncluded(t *testing.T) {
	SIZE_ID := getTestSizes()
	bidder := new(RubiconAdapter)

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					SIZE_ID[15],
					SIZE_ID[10],
				},
			},
			Ext: json.RawMessage(`{
				"bidder": {
					"zoneId": 8394,
					"siteId": 283282,
					"accountId": 7891,
					"inventory": {"key1" : "val1"},
					"visitor": {"key2" : "val2"}
				},
				"context": {
					"data": {
                        "adserver": {
                             "adslot": "/test-adslot",
                             "name": "gam"
                        }
					}
				}
			}`),
		}},
		App: &openrtb2.App{
			ID:   "com.test",
			Name: "testApp",
		},
	}

	reqs, _ := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	rubiconReq := &openrtb2.BidRequest{}
	if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
		t.Fatalf("Unexpected error while decoding request: %s", err)
	}

	assert.Equal(t, 1, len(rubiconReq.Imp),
		"Unexpected number of request impressions. Got %d. Expected %d", len(rubiconReq.Imp), 1)

	var rpImpExt rubiconImpExt
	if err := json.Unmarshal(rubiconReq.Imp[0].Ext, &rpImpExt); err != nil {
		t.Fatal("Error unmarshalling imp.ext")
	}
	assert.Equal(t, rpImpExt.GPID, "/test-adslot")
}

func TestOpenRTBFirstPartyDataPopulating(t *testing.T) {
	testScenarios := []rubiPopulateFpdAttributesScenario{
		{
			source: json.RawMessage(`{"sourceKey": ["sourceValue", "sourceValue2"]}`),
			target: map[string]interface{}{"targetKey": []interface{}{"targetValue"}},
			result: map[string]interface{}{"targetKey": []interface{}{"targetValue"}, "sourceKey": []interface{}{"sourceValue", "sourceValue2"}},
		},
		{
			source: json.RawMessage(`{"sourceKey": ["sourceValue", "sourceValue2"]}`),
			target: make(map[string]interface{}),
			result: map[string]interface{}{"sourceKey": []interface{}{"sourceValue", "sourceValue2"}},
		},
		{
			source: json.RawMessage(`{"sourceKey": "sourceValue"}`),
			target: make(map[string]interface{}),
			result: map[string]interface{}{"sourceKey": [1]string{"sourceValue"}},
		},
		{
			source: json.RawMessage(`{"sourceKey": true, "sourceKey2": [true, false, true]}`),
			target: make(map[string]interface{}),
			result: map[string]interface{}{"sourceKey": [1]string{"true"}, "sourceKey2": []string{"true", "false", "true"}},
		},
		{
			source: json.RawMessage(`{"sourceKey": 1, "sourceKey2": [1, 2, 3]}`),
			target: make(map[string]interface{}),
			result: map[string]interface{}{"sourceKey": [1]string{"1"}},
		},
		{
			source: json.RawMessage(`{"sourceKey": 1, "sourceKey2": 3.23}`),
			target: make(map[string]interface{}),
			result: map[string]interface{}{"sourceKey": [1]string{"1"}},
		},
		{
			source: json.RawMessage(`{"sourceKey": {}}`),
			target: make(map[string]interface{}),
			result: make(map[string]interface{}),
		},
	}

	for _, scenario := range testScenarios {
		populateFirstPartyDataAttributes(scenario.source, scenario.target)
		assert.Equal(t, scenario.result, scenario.target)
	}
}

func TestOpenRTBRequestWithBadvOverflowed(t *testing.T) {
	SIZE_ID := getTestSizes()
	bidder := new(RubiconAdapter)

	badvOverflowed := make([]string, 100)
	for i := range badvOverflowed {
		badvOverflowed[i] = strconv.Itoa(i)
	}

	request := &openrtb2.BidRequest{
		ID:   "test-request-id",
		BAdv: badvOverflowed,
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					SIZE_ID[15],
				},
			},
			Ext: json.RawMessage(`{
				"bidder": {
					"zoneId": 8394,
					"siteId": 283282,
					"accountId": 7891,
					"inventory": {"key1" : "val1"},
					"visitor": {"key2" : "val2"}
				}
			}`),
		}},
		App: &openrtb2.App{
			ID:   "com.test",
			Name: "testApp",
		},
	}

	reqs, _ := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	rubiconReq := &openrtb2.BidRequest{}
	if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
		t.Fatalf("Unexpected error while decoding request: %s", err)
	}

	badvRequest := rubiconReq.BAdv
	assert.Equal(t, badvOverflowed[:50], badvRequest, "Unexpected dfp_ad_unit_code: %s")
}

func TestOpenRTBRequestWithSpecificExtUserEids(t *testing.T) {
	SIZE_ID := getTestSizes()
	bidder := new(RubiconAdapter)

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					SIZE_ID[15],
					SIZE_ID[10],
				},
			},
			Ext: json.RawMessage(`{"bidder": {
				"zoneId": 8394,
				"siteId": 283282,
				"accountId": 7891
			}}`),
		}},
		App: &openrtb2.App{
			ID:   "com.test",
			Name: "testApp",
		},
		User: &openrtb2.User{
			Ext: json.RawMessage(`{"eids": [
			{
				"source": "pubcid",
				"id": "2402fc76-7b39-4f0e-bfc2-060ef7693648"
			},
			{
				"source": "adserver.org",
				"uids": [{
					"id": "3d50a262-bd8e-4be3-90b8-246291523907",
					"ext": {
						"rtiPartner": "TDID"
					}
				}]
			},
			{
				"source": "liveintent.com",
				"uids": [{
					"id": "T7JiRRvsRAmh88"
				}],
				"ext": {
					"segments": ["999","888"]
				}
			},
			{
				"source": "liveramp.com",
				"uids": [{
					"id": "LIVERAMPID"
				}],
				"ext": {
					"segments": ["111","222"]
				}
			}
			]}`),
		},
	}

	reqs, _ := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	rubiconReq := &openrtb2.BidRequest{}
	if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
		t.Fatalf("Unexpected error while decoding request: %s", err)
	}

	assert.NotNil(t, rubiconReq.User.Ext, "User.Ext object should not be nil.")

	var userExt rubiconUserExt
	if err := json.Unmarshal(rubiconReq.User.Ext, &userExt); err != nil {
		t.Fatal("Error unmarshalling request.user.ext object.")
	}

	assert.NotNil(t, userExt.Eids)
	assert.Equal(t, 4, len(userExt.Eids), "Eids values are not as expected!")

	assert.NotNil(t, userExt.TpID)
	assert.Equal(t, 2, len(userExt.TpID), "TpID values are not as expected!")

	// adserver.org
	assert.Equal(t, "tdid", userExt.TpID[0].Source, "TpID source value is not as expected!")

	// liveintent.com
	assert.Equal(t, "liveintent.com", userExt.TpID[1].Source, "TpID source value is not as expected!")

	// liveramp.com
	assert.Equal(t, "LIVERAMPID", userExt.LiverampIdl, "Liveramp_idl value is not as expected!")

	userExtRPTarget := make(map[string]interface{})
	if err := json.Unmarshal(userExt.RP.Target, &userExtRPTarget); err != nil {
		t.Fatal("Error unmarshalling request.user.ext.rp.target object.")
	}

	assert.Contains(t, userExtRPTarget, "LIseg", "request.user.ext.rp.target value is not as expected!")
	assert.Contains(t, userExtRPTarget["LIseg"], "888", "No segment with 888 as expected!")
	assert.Contains(t, userExtRPTarget["LIseg"], "999", "No segment with 999 as expected!")
}

func TestOpenRTBRequestWithVideoImpEvenIfImpHasBannerButAllRequiredVideoFields(t *testing.T) {
	SIZE_ID := getTestSizes()
	bidder := new(RubiconAdapter)

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					SIZE_ID[15],
					SIZE_ID[10],
				},
			},
			Video: &openrtb2.Video{
				W:           640,
				H:           360,
				MIMEs:       []string{"video/mp4"},
				Protocols:   []openrtb2.Protocol{openrtb2.ProtocolVAST10},
				MaxDuration: 30,
				Linearity:   1,
				API:         []openrtb2.APIFramework{},
			},
			Ext: json.RawMessage(`{"bidder": {
				"zoneId": 8394,
				"siteId": 283282,
				"accountId": 7891,
				"inventory": {"key1": "val1"},
				"visitor": {"key2": "val2"},
				"video": {"size_id": 1}
			}}`),
		}},
		App: &openrtb2.App{
			ID:   "com.test",
			Name: "testApp",
		},
	}

	reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs, "Got unexpected errors while building HTTP requests: %v", errs)

	assert.Equal(t, 1, len(reqs), "Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 1)

	rubiconReq := &openrtb2.BidRequest{}
	if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
		t.Fatalf("Unexpected error while decoding request: %s", err)
	}

	assert.Equal(t, 1, len(rubiconReq.Imp),
		"Unexpected number of request impressions. Got %d. Expected %d", len(rubiconReq.Imp), 1)

	assert.Nil(t, rubiconReq.Imp[0].Banner, "Unexpected banner object in request impression")

	assert.NotNil(t, rubiconReq.Imp[0].Video, "Video object must be in request impression")
}

func TestOpenRTBRequestWithVideoImpAndEnabledRewardedInventoryFlag(t *testing.T) {
	bidder := new(RubiconAdapter)

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Video: &openrtb2.Video{
				W:           640,
				H:           360,
				MIMEs:       []string{"video/mp4"},
				Protocols:   []openrtb2.Protocol{openrtb2.ProtocolVAST10},
				MaxDuration: 30,
				Linearity:   1,
				API:         []openrtb2.APIFramework{},
			},
			Ext: json.RawMessage(`{
			"prebid":{
				"is_rewarded_inventory": 1
			},
			"bidder": {
				"video": {"size_id": 1}
			}}`),
		}},
		App: &openrtb2.App{
			ID:   "com.test",
			Name: "testApp",
		},
	}

	reqs, _ := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	rubiconReq := &openrtb2.BidRequest{}
	if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
		t.Fatalf("Unexpected error while decoding request: %s", err)
	}

	videoExt := &rubiconVideoExt{}
	if err := json.Unmarshal(rubiconReq.Imp[0].Video.Ext, &videoExt); err != nil {
		t.Fatal("Error unmarshalling request.imp[i].video.ext object.")
	}

	assert.Equal(t, "rewarded", videoExt.VideoType,
		"Unexpected VideoType. Got %s. Expected %s", videoExt.VideoType, "rewarded")
}

func TestOpenRTBEmptyResponse(t *testing.T) {
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusNoContent,
	}
	bidder := new(RubiconAdapter)
	bidResponse, errs := bidder.MakeBids(nil, nil, httpResp)

	assert.Nil(t, bidResponse, "Expected empty response")
	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))
}

func TestOpenRTBSurpriseResponse(t *testing.T) {
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusAccepted,
	}
	bidder := new(RubiconAdapter)
	bidResponse, errs := bidder.MakeBids(nil, nil, httpResp)

	assert.Nil(t, bidResponse, "Expected empty response")

	assert.Equal(t, 1, len(errs), "Expected 1 error. Got %d", len(errs))
}

func TestOpenRTBStandardResponse(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{{
					W: 320,
					H: 50,
				}},
			},
			Ext: json.RawMessage(`{"bidder": {
				"accountId": 2763,
				"siteId": 68780,
				"zoneId": 327642
			}}`),
		}},
	}

	requestJson, _ := json.Marshal(request)
	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     "test-uri",
		Body:    requestJson,
		Headers: nil,
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"test-request-id","seatbid":[{"bid":[{"id":"1234567890","impid":"test-imp-id","price": 2,"crid":"4122982","adm":"some ad","h": 50,"w": 320,"ext":{"bidder":{"rp":{"targeting": {"key": "rpfl_2763", "values":["43_tier0100"]},"mime": "text/html","size_id": 43}}}}]}]}`),
	}

	bidder := new(RubiconAdapter)
	bidResponse, errs := bidder.MakeBids(request, reqData, httpResp)

	assert.NotNil(t, bidResponse, "Expected not empty response")
	assert.Equal(t, 1, len(bidResponse.Bids), "Expected 1 bid. Got %d", len(bidResponse.Bids))

	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))

	assert.Equal(t, openrtb_ext.BidTypeBanner, bidResponse.Bids[0].BidType,
		"Expected a banner bid. Got: %s", bidResponse.Bids[0].BidType)

	theBid := bidResponse.Bids[0].Bid
	assert.Equal(t, "1234567890", theBid.ID, "Bad bid ID. Expected %s, got %s", "1234567890", theBid.ID)
}

func TestOpenRTBResponseOverridePriceFromBidRequest(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{{
					W: 320,
					H: 50,
				}},
			},
			Ext: json.RawMessage(`{"bidder": {
				"accountId": 2763,
				"siteId": 68780,
				"zoneId": 327642
			}}`),
		}},
		Ext: json.RawMessage(`{"prebid": {
			"bidders": {
				"rubicon": {
					"debug": {
						"cpmoverride": 10
			}}}}}`),
	}

	requestJson, _ := json.Marshal(request)
	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     "test-uri",
		Body:    requestJson,
		Headers: nil,
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"test-request-id","seatbid":[{"bid":[{"id":"1234567890","impid":"test-imp-id","price": 2,"crid":"4122982","adm":"some ad","h": 50,"w": 320,"ext":{"bidder":{"rp":{"targeting": {"key": "rpfl_2763", "values":["43_tier0100"]},"mime": "text/html","size_id": 43}}}}]}]}`),
	}

	bidder := new(RubiconAdapter)
	bidResponse, errs := bidder.MakeBids(request, reqData, httpResp)

	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))

	assert.Equal(t, float64(10), bidResponse.Bids[0].Bid.Price,
		"Expected Price 10. Got: %s", bidResponse.Bids[0].Bid.Price)
}

func TestOpenRTBResponseSettingOfNetworkId(t *testing.T) {
	testScenarios := []rubiSetNetworkIdTestScenario{
		{
			bidExt:            nil,
			buyer:             "1",
			expectedNetworkId: 1,
			isNetworkIdSet:    true,
		},
		{
			bidExt:            nil,
			buyer:             "0",
			expectedNetworkId: 0,
			isNetworkIdSet:    false,
		},
		{
			bidExt:            nil,
			buyer:             "-1",
			expectedNetworkId: 0,
			isNetworkIdSet:    false,
		},
		{
			bidExt:            nil,
			buyer:             "1.1",
			expectedNetworkId: 0,
			isNetworkIdSet:    false,
		},
		{
			bidExt:            &openrtb_ext.ExtBidPrebid{},
			buyer:             "2",
			expectedNetworkId: 2,
			isNetworkIdSet:    true,
		},
		{
			bidExt:            &openrtb_ext.ExtBidPrebid{Meta: &openrtb_ext.ExtBidPrebidMeta{}},
			buyer:             "3",
			expectedNetworkId: 3,
			isNetworkIdSet:    true,
		},
		{
			bidExt:            &openrtb_ext.ExtBidPrebid{Meta: &openrtb_ext.ExtBidPrebidMeta{NetworkID: 5}},
			buyer:             "4",
			expectedNetworkId: 4,
			isNetworkIdSet:    true,
		},
		{
			bidExt:            &openrtb_ext.ExtBidPrebid{Meta: &openrtb_ext.ExtBidPrebidMeta{NetworkID: 5}},
			buyer:             "-1",
			expectedNetworkId: 5,
			isNetworkIdSet:    false,
		},
	}

	for _, scenario := range testScenarios {
		request := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{{
				ID:     "test-imp-id",
				Banner: &openrtb2.Banner{},
			}},
		}

		requestJson, _ := json.Marshal(request)
		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     "test-uri",
			Body:    requestJson,
			Headers: nil,
		}

		var givenBidExt json.RawMessage
		if scenario.bidExt != nil {
			marshalledExt, _ := json.Marshal(scenario.bidExt)
			givenBidExt = marshalledExt
		} else {
			givenBidExt = nil
		}
		givenBidResponse := rubiconBidResponse{
			SeatBid: []rubiconSeatBid{{Buyer: scenario.buyer,
				SeatBid: openrtb2.SeatBid{
					Bid: []openrtb2.Bid{{Price: 123.2, ImpID: "test-imp-id", Ext: givenBidExt}}}}},
		}
		body, _ := json.Marshal(&givenBidResponse)
		httpResp := &adapters.ResponseData{
			StatusCode: http.StatusOK,
			Body:       body,
		}

		bidder := new(RubiconAdapter)
		bidResponse, errs := bidder.MakeBids(request, reqData, httpResp)
		assert.Empty(t, errs)
		if scenario.isNetworkIdSet {
			networkdId, err := jsonparser.GetInt(bidResponse.Bids[0].Bid.Ext, "prebid", "meta", "networkId")
			assert.NoError(t, err)
			assert.Equal(t, scenario.expectedNetworkId, networkdId)
		} else {
			assert.Equal(t, bidResponse.Bids[0].Bid.Ext, givenBidExt)
		}
	}
}

func TestOpenRTBResponseOverridePriceFromCorrespondingImp(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{{
					W: 320,
					H: 50,
				}},
			},
			Ext: json.RawMessage(`{"bidder": {
				"accountId": 2763,
				"siteId": 68780,
				"zoneId": 327642,
				"debug": {
					"cpmoverride" : 20 
				}
			}}`),
		}},
		Ext: json.RawMessage(`{"prebid": {
			"bidders": {
				"rubicon": {
					"debug": {
						"cpmoverride": 10
			}}}}}`),
	}

	requestJson, _ := json.Marshal(request)
	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     "test-uri",
		Body:    requestJson,
		Headers: nil,
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"test-request-id","seatbid":[{"bid":[{"id":"1234567890","impid":"test-imp-id","price": 2,"crid":"4122982","adm":"some ad","h": 50,"w": 320,"ext":{"bidder":{"rp":{"targeting": {"key": "rpfl_2763", "values":["43_tier0100"]},"mime": "text/html","size_id": 43}}}}]}]}`),
	}

	bidder := new(RubiconAdapter)
	bidResponse, errs := bidder.MakeBids(request, reqData, httpResp)

	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))

	assert.Equal(t, float64(20), bidResponse.Bids[0].Bid.Price,
		"Expected Price 20. Got: %s", bidResponse.Bids[0].Bid.Price)
}

func TestOpenRTBCopyBidIdFromResponseIfZero(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID:  "test-request-id",
		Imp: []openrtb2.Imp{{}},
	}

	requestJson, _ := json.Marshal(request)
	reqData := &adapters.RequestData{Body: requestJson}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"test-request-id","bidid":"1234567890","seatbid":[{"bid":[{"id":"0","price": 1}]}]}`),
	}

	bidder := new(RubiconAdapter)
	bidResponse, _ := bidder.MakeBids(request, reqData, httpResp)

	theBid := bidResponse.Bids[0].Bid
	assert.Equal(t, "1234567890", theBid.ID, "Bad bid ID. Expected %s, got %s", "1234567890", theBid.ID)
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderRubicon, config.Adapter{
		Endpoint: "uri",
		XAPI: config.AdapterXAPI{
			Username: "xuser",
			Password: "xpass",
			Tracker:  "pbs-test-tracker",
		}})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "rubicontest", bidder)
}
