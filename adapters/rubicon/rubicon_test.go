package rubicon

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
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

func TestSetImpNative(t *testing.T) {
	testScenarios := []struct {
		request       string
		impNative     map[string]interface{}
		expectedError error
	}{
		{
			request:       "{}",
			impNative:     map[string]interface{}{"somekey": "someValue"},
			expectedError: fmt.Errorf("unable to find imp in json data"),
		},
		{
			request:       "{\"imp\":[]}",
			impNative:     map[string]interface{}{"somekey": "someValue"},
			expectedError: fmt.Errorf("unable to find imp[0] in json data"),
		},
		{
			request:       "{\"imp\":[{}]}",
			impNative:     map[string]interface{}{"somekey": "someValue"},
			expectedError: fmt.Errorf("unable to find imp[0].native in json data"),
		},
	}
	for _, scenario := range testScenarios {
		_, err := setImpNative([]byte(scenario.request), scenario.impNative)
		assert.Equal(t, scenario.expectedError, err)
	}
}

func TestResolveNativeObject(t *testing.T) {
	testScenarios := []struct {
		nativeObject  openrtb2.Native
		target        map[string]interface{}
		expectedError error
	}{
		{
			nativeObject:  openrtb2.Native{Ver: "1.0", Request: "{\"eventtrackers\": \"someWrongValue\"}"},
			target:        map[string]interface{}{},
			expectedError: nil,
		},
		{
			nativeObject:  openrtb2.Native{Ver: "1.1", Request: "{\"eventtrackers\": \"someWrongValue\"}"},
			target:        map[string]interface{}{},
			expectedError: nil,
		},
		{
			nativeObject:  openrtb2.Native{Ver: "1", Request: "{\"eventtrackers\": \"someWrongValue\"}"},
			target:        map[string]interface{}{},
			expectedError: fmt.Errorf("Eventtrackers are not present or not of array type"),
		},
		{
			nativeObject:  openrtb2.Native{Ver: "1", Request: "{\"eventtrackers\": [], \"context\": \"someWrongValue\"}"},
			target:        map[string]interface{}{},
			expectedError: fmt.Errorf("Context is not of int type"),
		},
		{
			nativeObject:  openrtb2.Native{Ver: "1", Request: "{\"eventtrackers\": [], \"plcmttype\": 2}"},
			target:        map[string]interface{}{},
			expectedError: nil,
		},
		{
			nativeObject:  openrtb2.Native{Ver: "1", Request: "{\"eventtrackers\": [], \"context\": 1}"},
			target:        map[string]interface{}{},
			expectedError: fmt.Errorf("Plcmttype is not present or not of int type"),
		},
	}
	for _, scenario := range testScenarios {
		_, err := resolveNativeObject(&scenario.nativeObject, scenario.target)
		assert.Equal(t, scenario.expectedError, err)
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
			bidFloor:         0,
			bidFloorCur:      "EUR",
			setMock:          func(m *mock.Mock) {},
			expectedBidFloor: 0,
			expectedBidCur:   "USD",
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

		bidder := new(RubiconAdapter)

		request := &openrtb2.BidRequest{
			ID: "test-request-id",
			Imp: []openrtb2.Imp{{
				ID:          "test-imp-id",
				BidFloorCur: scenario.bidFloorCur,
				BidFloor:    scenario.bidFloor,
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 300, H: 250},
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

func (m *mockCurrencyConversion) GetRate(from string, to string) (float64, error) {
	args := m.Called(from, to)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockCurrencyConversion) GetRates() *map[string]map[string]float64 {
	args := m.Called()
	return args.Get(0).(*map[string]map[string]float64)
}

func TestOpenRTBRequest(t *testing.T) {
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
					{W: 300, H: 250},
					{W: 300, H: 600},
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
				W:           ptrutil.ToPtr[int64](640),
				H:           ptrutil.ToPtr[int64](360),
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
					"uids": [{"id": "2402fc76-7b39-4f0e-bfc2-060ef7693648"}]
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

			assert.Equal(t, ptrutil.ToPtr[int64](640), rpRequest.Imp[0].Video.W,
				"Video width does not match. Expected %d, Got %d", 640, rpRequest.Imp[0].Video.W)

			assert.Equal(t, ptrutil.ToPtr[int64](360), rpRequest.Imp[0].Video.H,
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
		assert.Contains(t, userExt.Eids, openrtb2.EID{Source: "pubcid", UIDs: []openrtb2.UID{{ID: "2402fc76-7b39-4f0e-bfc2-060ef7693648"}}})
	}
}

func TestOpenRTBRequestWithBannerImpEvenIfImpHasVideo(t *testing.T) {
	bidder := new(RubiconAdapter)

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					{W: 300, H: 250},
				},
			},
			Video: &openrtb2.Video{
				W:     ptrutil.ToPtr[int64](640),
				H:     ptrutil.ToPtr[int64](360),
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
	bidder := new(RubiconAdapter)

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					{W: 300, H: 250},
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
				"data": {
					"adserver": {
						 "adslot": "/test-adslot",
						 "name": "gam"
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

	dfpAdUnitCode, err := jsonparser.GetString(rpImpExt.RP.Target, "dfp_ad_unit_code")
	if err != nil {
		t.Fatal("Error extracting dfp_ad_unit_code")
	}
	assert.Equal(t, dfpAdUnitCode, "/test-adslot")
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

func TestPbsHostInfoPopulating(t *testing.T) {
	bidder := RubiconAdapter{
		URI:          "url",
		externalURI:  "externalUrl",
		XAPIUsername: "username",
		XAPIPassword: "password",
	}

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					{W: 300, H: 250},
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

	var rpImpExt rubiconImpExt
	if err := json.Unmarshal(rubiconReq.Imp[0].Ext, &rpImpExt); err != nil {
		t.Fatalf("Error unmarshalling imp.ext: %s", err)
	}

	var pbsLogin string
	pbsLogin, err := jsonparser.GetString(rpImpExt.RP.Target, "pbs_login")
	if err != nil {
		t.Fatal("Error extracting pbs_login")
	}
	assert.Equal(t, pbsLogin, "username", "Unexpected pbs_login value")

	var pbsVersion string
	pbsVersion, err = jsonparser.GetString(rpImpExt.RP.Target, "pbs_version")
	if err != nil {
		t.Fatal("Error extracting pbs_version")
	}
	assert.Equal(t, pbsVersion, "", "Unexpected pbs_version value")

	var pbsUrl string
	pbsUrl, err = jsonparser.GetString(rpImpExt.RP.Target, "pbs_url")
	if err != nil {
		t.Fatal("Error extracting pbs_url")
	}
	assert.Equal(t, pbsUrl, "externalUrl", "Unexpected pbs_url value")
}

func TestOpenRTBRequestWithBadvOverflowed(t *testing.T) {
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
					{W: 300, H: 250},
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

func TestOpenRTBRequestWithVideoImpEvenIfImpHasBannerButAllRequiredVideoFields(t *testing.T) {
	bidder := new(RubiconAdapter)

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					{W: 300, H: 250},
				},
			},
			Video: &openrtb2.Video{
				W:           ptrutil.ToPtr[int64](640),
				H:           ptrutil.ToPtr[int64](360),
				MIMEs:       []string{"video/mp4"},
				Protocols:   []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST10},
				MaxDuration: 30,
				Linearity:   1,
				API:         []adcom1.APIFramework{},
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
				W:           ptrutil.ToPtr[int64](640),
				H:           ptrutil.ToPtr[int64](360),
				MIMEs:       []string{"video/mp4"},
				Protocols:   []adcom1.MediaCreativeSubtype{adcom1.CreativeVAST10},
				MaxDuration: 30,
				Linearity:   1,
				API:         []adcom1.APIFramework{},
			},
			Ext: json.RawMessage(`{
			"prebid":{
				"is_rewarded_inventory": 1
			},
			"bidder": {
				"video": {"size_id": 1},
				"zoneId": "123",
				"siteId": 1234,
				"accountId": "444"
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
			marshalledExt, _ := json.Marshal(&openrtb_ext.ExtBid{Prebid: scenario.bidExt})
			givenBidExt = marshalledExt
		} else {
			givenBidExt = nil
		}

		givenBidResponse := rubiconBidResponse{
			SeatBid: []rubiconSeatBid{{Buyer: scenario.buyer,
				Bid: []rubiconBid{{
					Bid: openrtb2.Bid{Price: 123.2, ImpID: "test-imp-id", Ext: givenBidExt}}}}},
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

func TestUpdateBidExtWithMeta_OnlySeatSet(t *testing.T) {
	bid := rubiconBid{
		Bid: openrtb2.Bid{
			Ext: nil,
		},
		AdmNative: nil,
	}
	seat := "test-seat"
	buyer := 0

	ext := updateBidExtWithMeta(bid, buyer, seat)
	assert.NotNil(t, ext, "Expected non-nil ext when seat is set")

	var extPrebidWrapper struct {
		Prebid struct {
			Meta struct {
				Seat      string `json:"seat"`
				NetworkID *int   `json:"networkId,omitempty"`
			} `json:"meta"`
		} `json:"prebid"`
	}
	err := json.Unmarshal(ext, &extPrebidWrapper)
	assert.NoError(t, err, "Unmarshal should succeed")
	assert.Equal(t, seat, extPrebidWrapper.Prebid.Meta.Seat, "Seat should be set")
	assert.Zero(t, extPrebidWrapper.Prebid.Meta.NetworkID, "NetworkID should be omitted or zero")
}

func TestOpenRTBResponseBidExtPrebidMetaPassthrough(t *testing.T) {
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

	bidExt := &openrtb_ext.ExtBid{Prebid: &openrtb_ext.ExtBidPrebid{Meta: &openrtb_ext.ExtBidPrebidMeta{AdapterCode: "1", MediaType: "banner"}}}
	givenBidExt, _ := json.Marshal(bidExt)

	givenBidResponse := rubiconBidResponse{
		SeatBid: []rubiconSeatBid{{
			Bid: []rubiconBid{{
				Bid: openrtb2.Bid{Price: 123.2, ImpID: "test-imp-id", Ext: givenBidExt}}}}},
	}
	body, _ := json.Marshal(&givenBidResponse)
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       body,
	}

	bidder := new(RubiconAdapter)
	bidResponse, errs := bidder.MakeBids(request, reqData, httpResp)
	assert.Empty(t, errs)

	var actualBidExt openrtb_ext.ExtBid
	err := json.Unmarshal(bidResponse.Bids[0].Bid.Ext, &actualBidExt)
	assert.NoError(t, err)
	assert.Equal(t, bidExt.Prebid.Meta, actualBidExt.Prebid.Meta)
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
		}}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "rubicontest", bidder)
}
