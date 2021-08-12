package sharethrough

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

type MockUtil struct {
	mockCanAutoPlayVideo func() bool
	mockGdprApplies      func() bool
	mockGetPlacementSize func() (int64, int64)
	mockParseUserInfo    func() userInfo
	UtilityInterface
}

func (m MockUtil) canAutoPlayVideo(userAgent string, parsers UserAgentParsers) bool {
	return m.mockCanAutoPlayVideo()
}

func (m MockUtil) gdprApplies(request *openrtb2.BidRequest) bool {
	return m.mockGdprApplies()
}

func (m MockUtil) getPlacementSize(imp openrtb2.Imp, strImpParams openrtb_ext.ExtImpSharethrough) (height, width int64) {
	return m.mockGetPlacementSize()
}

func (m MockUtil) parseUserInfo(user *openrtb2.User) (ui userInfo) {
	return m.mockParseUserInfo()
}

func (m MockUtil) getClock() ClockInterface {
	return MockClock{}
}

type MockClock struct {
	ClockInterface
}

func (m MockClock) now() time.Time {
	return time.Date(2019, 9, 12, 11, 29, 0, 123456, time.UTC)
}

func assertRequestDataEquals(t *testing.T, testName string, expected *adapters.RequestData, actual *adapters.RequestData) {
	t.Logf("Test case: %s\n", testName)
	if expected.Method != actual.Method {
		t.Errorf("Method mismatch: expected %s got %s\n", expected.Method, actual.Method)
	}
	if expected.Uri != actual.Uri {
		t.Errorf("Uri mismatch: expected %s got %s\n", expected.Uri, actual.Uri)
	}
	if len(expected.Body) != len(actual.Body) {
		t.Errorf("Body mismatch: expected %s got %s\n", expected.Body, actual.Body)
	}
	if len(expected.Headers) != len(actual.Headers) {
		t.Errorf("Number of headers mismatch: expected %d got %d\n", len(expected.Headers), len(actual.Headers))
	}
	for headerIndex, expectedHeader := range expected.Headers {
		if expectedHeader[0] != actual.Headers[headerIndex][0] {
			t.Errorf("Header %s mismatch: expected %s got %s\n", headerIndex, expectedHeader[0], actual.Headers[headerIndex][0])
		}
	}
}

func TestSuccessRequestFromOpenRTB(t *testing.T) {
	tests := map[string]struct {
		inputImp openrtb2.Imp
		inputReq *openrtb2.BidRequest
		inputDom string
		expected *adapters.RequestData
	}{
		"Generates the correct AdServer request from Imp (no user provided)": {
			inputImp: openrtb2.Imp{
				ID:  "abc",
				Ext: []byte(`{ "bidder": {"pkey": "pkey", "iframe": true, "iframeSize": [10, 20], "bidfloor": 1.0, "data": { "pbadslot": "adslot" } } }`),
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{{H: 30, W: 40}},
				},
			},
			inputReq: &openrtb2.BidRequest{
				App: &openrtb2.App{Ext: []byte(`{}`)},
				Device: &openrtb2.Device{
					UA: "Android Chome/60",
					IP: "127.0.0.1",
				},
				Site: &openrtb2.Site{Page: "http://a.domain.com/page"},
				BAdv: []string{"domain1.com", "domain2.com"},
				TMax: 700,
			},
			inputDom: "http://a.domain.com",
			expected: &adapters.RequestData{
				Method: "POST",
				Uri:    "http://abc.com",
				Body:   []byte(`{"badv":["domain1.com","domain2.com"],"tmax":700,"deadline":"2019-09-12T11:29:00.700123456Z","bidfloor":1}`),
				Headers: http.Header{
					"Content-Type":    []string{"application/json;charset=utf-8"},
					"Accept":          []string{"application/json"},
					"Origin":          []string{"http://a.domain.com"},
					"Referer":         []string{"http://a.domain.com/page"},
					"User-Agent":      []string{"Android Chome/60"},
					"X-Forwarded-For": []string{"127.0.0.1"},
				},
			},
		},
		"Generates width/height if not provided": {
			inputImp: openrtb2.Imp{
				ID:  "abc",
				Ext: []byte(`{ "bidder": {"pkey": "pkey", "iframe": true} }`),
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{{H: 30, W: 40}},
				},
			},
			inputReq: &openrtb2.BidRequest{
				App: &openrtb2.App{Ext: []byte(`{}`)},
				Device: &openrtb2.Device{
					UA: "Android Chome/60",
					IP: "127.0.0.1",
				},
				Site: &openrtb2.Site{Page: "http://a.domain.com/page"},
				BAdv: []string{"domain1.com", "domain2.com"},
				TMax: 700,
			},
			inputDom: "http://a.domain.com",
			expected: &adapters.RequestData{
				Method: "POST",
				Uri:    "http://abc.com",
				Body:   []byte(`{"badv":["domain1.com","domain2.com"],"tmax":700,"deadline":"2019-09-12T11:29:00.700123456Z"}`),
				Headers: http.Header{
					"Content-Type":    []string{"application/json;charset=utf-8"},
					"Accept":          []string{"application/json"},
					"Origin":          []string{"http://a.domain.com"},
					"Referer":         []string{"http://a.domain.com/page"},
					"User-Agent":      []string{"Android Chome/60"},
					"X-Forwarded-For": []string{"127.0.0.1"},
				},
			},
		},
	}

	mockUriHelper := MockStrUriHelper{
		mockBuildUri: func() string {
			return "http://abc.com"
		},
	}

	mockUtil := MockUtil{
		mockCanAutoPlayVideo: func() bool { return true },
		mockGdprApplies:      func() bool { return true },
		mockGetPlacementSize: func() (int64, int64) { return 100, 200 },
		mockParseUserInfo:    func() userInfo { return userInfo{Consent: "ok", TtdUid: "ttduid", StxUid: "stxuid"} },
	}

	adServer := StrOpenRTBTranslator{UriHelper: mockUriHelper, Util: mockUtil, UserAgentParsers: UserAgentParsers{
		ChromeVersion:    regexp.MustCompile(`Chrome\/(?P<ChromeVersion>\d+)`),
		ChromeiOSVersion: regexp.MustCompile(`CriOS\/(?P<chromeiOSVersion>\d+)`),
		SafariVersion:    regexp.MustCompile(`Version\/(?P<safariVersion>\d+)`),
	}}
	for testName, test := range tests {
		outputSuccess, outputError := adServer.requestFromOpenRTB(test.inputImp, test.inputReq, test.inputDom)
		assertRequestDataEquals(t, testName, test.expected, outputSuccess)
		if outputError != nil {
			t.Errorf("Expected no errors, got %s\n", outputError)
		}
	}
}

func TestFailureRequestFromOpenRTB(t *testing.T) {
	tests := map[string]struct {
		inputImp      openrtb2.Imp
		inputReq      *openrtb2.BidRequest
		expectedError string
	}{
		"Fails when unable to parse imp.Ext": {
			inputImp: openrtb2.Imp{
				Ext: []byte(`{"abc`),
			},
			inputReq: &openrtb2.BidRequest{
				Device: &openrtb2.Device{UA: "A", IP: "ip"},
				Site:   &openrtb2.Site{Page: "page"},
			},
			expectedError: `unexpected end of JSON input`,
		},
		"Fails when unable to parse imp.Ext.Bidder": {
			inputImp: openrtb2.Imp{
				Ext: []byte(`{ "bidder": "{ abc" }`),
			},
			inputReq: &openrtb2.BidRequest{
				Device: &openrtb2.Device{UA: "A", IP: "ip"},
				Site:   &openrtb2.Site{Page: "page"},
			},
			expectedError: `json: cannot unmarshal string into Go value of type openrtb_ext.ExtImpSharethrough`,
		},
	}

	mockUriHelper := MockStrUriHelper{
		mockBuildUri: func() string {
			return "http://abc.com"
		},
	}

	mockUtil := MockUtil{
		mockCanAutoPlayVideo: func() bool { return true },
		mockGdprApplies:      func() bool { return true },
		mockGetPlacementSize: func() (int64, int64) { return 100, 200 },
		mockParseUserInfo:    func() userInfo { return userInfo{Consent: "ok", TtdUid: "ttduid", StxUid: "stxuid"} },
	}

	adServer := StrOpenRTBTranslator{UriHelper: mockUriHelper, Util: mockUtil, UserAgentParsers: UserAgentParsers{
		ChromeVersion:    regexp.MustCompile(`Chrome\/(?P<ChromeVersion>\d+)`),
		ChromeiOSVersion: regexp.MustCompile(`CriOS\/(?P<chromeiOSVersion>\d+)`),
		SafariVersion:    regexp.MustCompile(`Version\/(?P<safariVersion>\d+)`),
	}}

	assert := assert.New(t)
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		output, outputError := adServer.requestFromOpenRTB(test.inputImp, test.inputReq, "anything")

		assert.Nil(output)
		assert.NotNil(outputError)
		assert.Equal(test.expectedError, outputError.Error())
	}
}

func assertBidderResponseEquals(t *testing.T, testName string, expected adapters.BidderResponse, actual adapters.BidderResponse) {
	t.Logf("Test case: %s\n", testName)
	if len(expected.Bids) != len(actual.Bids) {
		t.Errorf("Expected %d bids in BidResponse, got %d\n", len(expected.Bids), len(actual.Bids))
		return
	}
	for index, expectedTypedBid := range expected.Bids {
		if expectedTypedBid.BidType != actual.Bids[index].BidType {
			t.Errorf("Bid[%d]: Type mismatch, expected %s got %s\n", index, expectedTypedBid.BidType, actual.Bids[index].BidType)
		}
		if expectedTypedBid.Bid.AdID != actual.Bids[index].Bid.AdID {
			t.Errorf("Bid[%d]: AdID mismatch, expected %s got %s\n", index, expectedTypedBid.Bid.AdID, actual.Bids[index].Bid.AdID)
		}
		if expectedTypedBid.Bid.ID != actual.Bids[index].Bid.ID {
			t.Errorf("Bid[%d]: ID mismatch, expected %s got %s\n", index, expectedTypedBid.Bid.ID, actual.Bids[index].Bid.ID)
		}
		if expectedTypedBid.Bid.ImpID != actual.Bids[index].Bid.ImpID {
			t.Errorf("Bid[%d]: ImpID mismatch, expected %s got %s\n", index, expectedTypedBid.Bid.ImpID, actual.Bids[index].Bid.ImpID)
		}
		if expectedTypedBid.Bid.Price != actual.Bids[index].Bid.Price {
			t.Errorf("Bid[%d]: Price mismatch, expected %f got %f\n", index, expectedTypedBid.Bid.Price, actual.Bids[index].Bid.Price)
		}
		if expectedTypedBid.Bid.CID != actual.Bids[index].Bid.CID {
			t.Errorf("Bid[%d]: CID mismatch, expected %s got %s\n", index, expectedTypedBid.Bid.CID, actual.Bids[index].Bid.CID)
		}
		if expectedTypedBid.Bid.CrID != actual.Bids[index].Bid.CrID {
			t.Errorf("Bid[%d]: CrID mismatch, expected %s got %s\n", index, expectedTypedBid.Bid.CrID, actual.Bids[index].Bid.CrID)
		}
		if expectedTypedBid.Bid.DealID != actual.Bids[index].Bid.DealID {
			t.Errorf("Bid[%d]: DealID mismatch, expected %s got %s\n", index, expectedTypedBid.Bid.DealID, actual.Bids[index].Bid.DealID)
		}
		if expectedTypedBid.Bid.H != actual.Bids[index].Bid.H {
			t.Errorf("Bid[%d]: H mismatch, expected %d got %d\n", index, expectedTypedBid.Bid.H, actual.Bids[index].Bid.H)
		}
		if expectedTypedBid.Bid.W != actual.Bids[index].Bid.W {
			t.Errorf("Bid[%d]: W mismatch, expected %d got %d\n", index, expectedTypedBid.Bid.W, actual.Bids[index].Bid.W)
		}
	}
}

func TestSuccessResponseToOpenRTB(t *testing.T) {
	tests := map[string]struct {
		inputButlerReq  *adapters.RequestData
		inputStrResp    []byte
		expectedSuccess *adapters.BidderResponse
		expectedErrors  []error
	}{
		"Generates expected openRTB bid response": {
			inputButlerReq: &adapters.RequestData{
				Uri: "http://uri.com?placement_key=pkey&bidId=bidid&height=20&width=30",
			},
			inputStrResp: []byte(`{ "adserverRequestId": "arid", "bidId": "bid", "creatives": [{"cpm": 10, "creative": {"campaign_key": "cmpKey", "creative_key": "creaKey", "deal_id": "dealId"}}] }`),
			expectedSuccess: &adapters.BidderResponse{
				Bids: []*adapters.TypedBid{{
					BidType: openrtb_ext.BidTypeBanner,
					Bid: &openrtb2.Bid{
						AdID:   "arid",
						ID:     "bid",
						ImpID:  "bidid",
						Price:  10,
						CID:    "cmpKey",
						CrID:   "creaKey",
						DealID: "dealId",
						H:      20,
						W:      30,
					},
				}},
			},
			expectedErrors: []error{},
		},
	}

	adServer := StrOpenRTBTranslator{Util: Util{Clock: MockClock{}}, UriHelper: StrUriHelper{}}
	for testName, test := range tests {
		outputSuccess, outputErrors := adServer.responseToOpenRTB(test.inputStrResp, test.inputButlerReq)
		assertBidderResponseEquals(t, testName, *test.expectedSuccess, *outputSuccess)
		if len(outputErrors) != len(test.expectedErrors) {
			t.Errorf("Expected %d errors, got %d\n", len(test.expectedErrors), len(outputErrors))
		}
	}
}

func TestFailResponseToOpenRTB(t *testing.T) {
	tests := map[string]struct {
		inputButlerReq  *adapters.RequestData
		inputStrResp    []byte
		expectedSuccess *adapters.BidderResponse
		expectedErrors  []error
	}{
		"Returns nil if no creatives provided": {
			inputButlerReq:  &adapters.RequestData{},
			inputStrResp:    []byte(`{}`),
			expectedSuccess: nil,
			expectedErrors: []error{
				&errortypes.BadInput{Message: "No creative provided"},
			},
		},
		"Returns nil if failed to parse Uri": {
			inputButlerReq: &adapters.RequestData{
				Uri: "wrong format url",
			},
			inputStrResp:    []byte(`{ "creatives": [{"creative": {}}] }`),
			expectedSuccess: nil,
			expectedErrors: []error{
				&errortypes.BadInput{Message: `strconv.ParseUint: parsing "": invalid syntax`},
			},
		},
		"Returns error if failed parsing body": {
			inputButlerReq:  &adapters.RequestData{},
			inputStrResp:    []byte(`{ wrong json`),
			expectedSuccess: nil,
			expectedErrors: []error{
				&errortypes.BadInput{Message: "Unable to parse response JSON"},
			},
		},
	}

	adServer := StrOpenRTBTranslator{UriHelper: StrUriHelper{}}
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		outputSuccess, outputErrors := adServer.responseToOpenRTB(test.inputStrResp, test.inputButlerReq)

		if test.expectedSuccess != outputSuccess {
			t.Errorf("Expected result %+v, got %+v\n", test.expectedSuccess, outputSuccess)
		}

		if len(outputErrors) != len(test.expectedErrors) {
			t.Errorf("Expected %d errors, got %d\n", len(test.expectedErrors), len(outputErrors))
		}

		for index, expectedError := range test.expectedErrors {
			if fmt.Sprintf("%T", expectedError) != fmt.Sprintf("%T", outputErrors[index]) {
				t.Errorf("Error type mismatch, expected %T, got %T\n", expectedError, outputErrors[index])
			}
			if expectedError.Error() != outputErrors[index].Error() {
				t.Errorf("Expected error %s, got %s\n", expectedError.Error(), outputErrors[index].Error())
			}
		}
	}
}

func TestBuildBody(t *testing.T) {
	tests := map[string]struct {
		inputRequest  *openrtb2.BidRequest
		inputImp      openrtb_ext.ExtImpSharethrough
		expectedJson  []byte
		expectedError error
	}{
		"Empty input: skips badomains, tmax default to 10 sec and sets deadline accordingly": {
			inputRequest:  &openrtb2.BidRequest{},
			inputImp:      openrtb_ext.ExtImpSharethrough{},
			expectedJson:  []byte(`{"tmax":10000, "deadline":"2019-09-12T11:29:10.000123456Z"}`),
			expectedError: nil,
		},
		"Sets badv as list of domains according to Badv (tmax default to 10 sec and sets deadline accordingly)": {
			inputRequest: &openrtb2.BidRequest{
				BAdv: []string{"dom1.com", "dom2.com"},
			},
			inputImp:      openrtb_ext.ExtImpSharethrough{},
			expectedJson:  []byte(`{"badv": ["dom1.com", "dom2.com"], "tmax":10000, "deadline":"2019-09-12T11:29:10.000123456Z"}`),
			expectedError: nil,
		},
		"Sets tmax and deadline according to Tmax": {
			inputRequest: &openrtb2.BidRequest{
				TMax: 500,
			},
			inputImp:      openrtb_ext.ExtImpSharethrough{},
			expectedJson:  []byte(`{"tmax": 500, "deadline":"2019-09-12T11:29:00.500123456Z"}`),
			expectedError: nil,
		},
		"Sets bidfloor according to the Imp object": {
			inputRequest: &openrtb2.BidRequest{},
			inputImp: openrtb_ext.ExtImpSharethrough{
				BidFloor: 1.23,
			},
			expectedJson:  []byte(`{"tmax":10000, "deadline":"2019-09-12T11:29:10.000123456Z", "bidfloor":1.23}`),
			expectedError: nil,
		},
	}

	assert := assert.New(t)
	helper := StrBodyHelper{Clock: MockClock{}}

	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		outputJson, outputError := helper.buildBody(test.inputRequest, test.inputImp)

		assert.JSONEq(string(test.expectedJson), string(outputJson))
		assert.Equal(test.expectedError, outputError)
	}
}

func TestBuildUri(t *testing.T) {
	tests := map[string]struct {
		inputParams StrAdSeverParams
		inputApp    *openrtb2.App
		expected    []string
	}{
		"Generates expected URL, appending all params": {
			inputParams: StrAdSeverParams{
				Pkey:               "pkey",
				BidID:              "bid",
				GPID:               "gpid",
				ConsentRequired:    true,
				ConsentString:      "consent",
				USPrivacySignal:    "ccpa",
				InstantPlayCapable: true,
				Iframe:             false,
				Height:             20,
				Width:              30,
				TheTradeDeskUserId: "ttd123",
				SharethroughUserId: "stx123",
			},
			expected: []string{
				"http://abc.com?",
				"placement_key=pkey",
				"bidId=bid",
				"gpid=gpid",
				"consent_required=true",
				"consent_string=consent",
				"us_privacy=ccpa",
				"instant_play_capable=true",
				"stayInIframe=false",
				"height=20",
				"width=30",
				"supplyId=FGMrCMMc",
				"strVersion=" + strconv.FormatInt(strVersion, 10),
				"ttduid=ttd123",
				"stxuid=stx123",
				"adRequestAt=2019-09-12T11%3A29%3A00.000123456Z",
			},
		},
	}

	uriHelper := StrUriHelper{BaseURI: "http://abc.com", Clock: MockClock{}}
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		output := uriHelper.buildUri(test.inputParams)

		for _, uriParam := range test.expected {
			if !strings.Contains(output, uriParam) {
				t.Errorf("Expected %s to be found in URL, got %s\n", uriParam, output)
			}
		}
	}
}

func assertStrAdServerParamsEquals(t *testing.T, testName string, expected *StrAdSeverParams, actual *StrAdSeverParams) {
	t.Logf("Test case: %s\n", testName)
	if expected.Pkey != actual.Pkey {
		t.Errorf("Expected Pkey to be %s, got %s\n", expected.Pkey, actual.Pkey)
	}
	if expected.BidID != actual.BidID {
		t.Errorf("Expected BidID to be %s, got %s\n", expected.BidID, actual.BidID)
	}
	if expected.Iframe != actual.Iframe {
		t.Errorf("Expected Iframe to be %t, got %t\n", expected.Iframe, actual.Iframe)
	}
	if expected.Height != actual.Height {
		t.Errorf("Expected Height to be %d, got %d\n", expected.Height, actual.Height)
	}
	if expected.Width != actual.Width {
		t.Errorf("Expected Width to be %d, got %d\n", expected.Width, actual.Width)
	}
	if expected.ConsentRequired != actual.ConsentRequired {
		t.Errorf("Expected ConsentRequired to be %t, got %t\n", expected.ConsentRequired, actual.ConsentRequired)
	}
	if expected.ConsentString != actual.ConsentString {
		t.Errorf("Expected ConsentString to be %s, got %s\n", expected.ConsentString, actual.ConsentString)
	}
}

func TestSuccessParseUri(t *testing.T) {
	tests := map[string]struct {
		input           string
		expectedSuccess *StrAdSeverParams
	}{
		"Decodes URI successfully": {
			input: "http://abc.com?placement_key=pkey&bidId=bid&consent_required=true&consent_string=consent&instant_play_capable=true&stayInIframe=false&height=20&width=30&hbVersion=1&supplyId=FGMrCMMc&strVersion=1.0.0",
			expectedSuccess: &StrAdSeverParams{
				Pkey:            "pkey",
				BidID:           "bid",
				Iframe:          false,
				Height:          20,
				Width:           30,
				ConsentRequired: true,
				ConsentString:   "consent",
			},
		},
	}

	uriHelper := StrUriHelper{}
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		output, actualError := uriHelper.parseUri(test.input)

		assertStrAdServerParamsEquals(t, testName, test.expectedSuccess, output)
		if actualError != nil {
			t.Errorf("Expected no errors, got %s\n", actualError)
		}
	}
}

func TestFailParseUri(t *testing.T) {
	tests := map[string]struct {
		input         string
		expectedError string
	}{
		"Fails decoding if unable to parse URI": {
			input:         "test:/#$%?#",
			expectedError: `parse (\")?test:/#\$%\?#(\")?: invalid URL escape \"%\?#\"`,
		},
		"Fails decoding if height not provided": {
			input:         "http://abc.com?width=10",
			expectedError: `strconv.ParseUint: parsing \"\": invalid syntax`,
		},
		"Fails decoding if width not provided": {
			input:         "http://abc.com?height=10",
			expectedError: `strconv.ParseUint: parsing \"\": invalid syntax`,
		},
	}

	assert := assert.New(t)

	uriHelper := StrUriHelper{}
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		output, actualError := uriHelper.parseUri(test.input)

		assert.Nil(output)
		assert.NotNil(actualError)
		assert.Regexp(test.expectedError, actualError.Error())
	}
}
