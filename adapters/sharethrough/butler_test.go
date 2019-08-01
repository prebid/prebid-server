package sharethrough

import (
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

type MockUtil struct {
	mockCanAutoPlayVideo  func() bool
	mockGdprApplies       func() bool
	mockGdprConsentString func() string
	mockGenerateHBUri     func() string
	mockGetPlacementSize  func() (uint64, uint64)
	UtilityInterface
}

func (m MockUtil) canAutoPlayVideo(userAgent string) bool {
	return m.mockCanAutoPlayVideo()
}

func (m MockUtil) gdprApplies(request *openrtb.BidRequest) bool {
	return m.mockGdprApplies()
}

func (m MockUtil) gdprConsentString(bidRequest *openrtb.BidRequest) string {
	return m.mockGdprConsentString()
}

func (m MockUtil) generateHBUri(baseUrl string, params StrAdSeverParams, app *openrtb.App) string {
	return m.mockGenerateHBUri()
}

func (m MockUtil) getPlacementSize(formats []openrtb.Format) (height uint64, width uint64) {
	return m.mockGetPlacementSize()
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
		inputImp openrtb.Imp
		inputReq *openrtb.BidRequest
		inputDom string
		expected *adapters.RequestData
	}{
		"Generates the correct AdServer request from Imp": {
			inputImp: openrtb.Imp{
				ID:  "abc",
				Ext: []byte(`{ "bidder": {"pkey": "pkey", "iframe": true, "iframeSize": [10, 20]} }`),
				Banner: &openrtb.Banner{
					Format: []openrtb.Format{{H: 30, W: 40}},
				},
			},
			inputReq: &openrtb.BidRequest{
				App: &openrtb.App{Ext: []byte(`{}`)},
				Device: &openrtb.Device{
					UA: "Android Chome/60",
					IP: "127.0.0.1",
				},
			},
			inputDom: "http://a.domain.com",
			expected: &adapters.RequestData{
				Method: "POST",
				Uri:    "http://abc.com",
				Body:   nil,
				Headers: http.Header{
					"Content-Type":    []string{"text/plain;charset=utf-8"},
					"Accept":          []string{"application/json"},
					"Origin":          []string{"http://a.domain.com"},
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

	adServer := StrOpenRTBTranslator{UriHelper: mockUriHelper, Util: Util{}, UserAgentParsers: UserAgentParsers{
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
		inputStrResp    openrtb_ext.ExtImpSharethroughResponse
		expectedSuccess *adapters.BidderResponse
		expectedErrors  []error
	}{
		"Generates expected openRTB bid response": {
			inputButlerReq: &adapters.RequestData{
				Uri: "http://uri.com?placement_key=pkey&bidId=bidid&height=20&width=30",
			},
			inputStrResp: openrtb_ext.ExtImpSharethroughResponse{
				AdServerRequestID: "arid",
				BidID:             "bid",
				Creatives: []openrtb_ext.ExtImpSharethroughCreative{{
					CPM: 10,
					Metadata: openrtb_ext.ExtImpSharethroughCreativeMetadata{
						CampaignKey: "cmpKey",
						CreativeKey: "creaKey",
						DealID:      "dealId",
					},
				}},
			},
			expectedSuccess: &adapters.BidderResponse{
				Bids: []*adapters.TypedBid{{
					BidType: openrtb_ext.BidTypeNative,
					Bid: &openrtb.Bid{
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

	adServer := StrOpenRTBTranslator{Util: Util{}, UriHelper: StrUriHelper{}}
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
		inputStrResp    openrtb_ext.ExtImpSharethroughResponse
		expectedSuccess *adapters.BidderResponse
		expectedErrors  []error
	}{
		"Returns nil if no creatives provided": {
			inputButlerReq: &adapters.RequestData{},
			inputStrResp: openrtb_ext.ExtImpSharethroughResponse{
				Creatives: []openrtb_ext.ExtImpSharethroughCreative{},
			},
			expectedSuccess: nil,
			expectedErrors: []error{
				&errortypes.BadInput{Message: "No creative provided"},
			},
		},
		"Returns nil if failed to parse Uri": {
			inputButlerReq: &adapters.RequestData{
				Uri: "wrong format url",
			},
			inputStrResp: openrtb_ext.ExtImpSharethroughResponse{
				Creatives: []openrtb_ext.ExtImpSharethroughCreative{{}},
			},
			expectedSuccess: nil,
			expectedErrors: []error{
				&errortypes.BadInput{Message: `strconv.ParseUint: parsing "": invalid syntax`},
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

func TestBuildUri(t *testing.T) {
	tests := map[string]struct {
		inputParams StrAdSeverParams
		inputApp    *openrtb.App
		expected    []string
	}{
		"Generates expected URL, appending all params": {
			inputParams: StrAdSeverParams{
				Pkey:               "pkey",
				BidID:              "bid",
				ConsentRequired:    true,
				ConsentString:      "consent",
				InstantPlayCapable: true,
				Iframe:             false,
				Height:             20,
				Width:              30,
				TheTradeDeskUserId: "ttd123",
			},
			expected: []string{
				"http://abc.com?",
				"placement_key=pkey",
				"bidId=bid",
				"consent_required=true",
				"consent_string=consent",
				"instant_play_capable=true",
				"stayInIframe=false",
				"height=20",
				"width=30",
				"supplyId=FGMrCMMc",
				"strVersion=" + strVersion,
				"ttduid=ttd123",
			},
		},
	}

	uriHelper := StrUriHelper{BaseURI: "http://abc.com"}
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
			input:         "wrong URI",
			expectedError: `strconv.ParseUint: parsing "": invalid syntax`,
		},
		"Fails decoding if height not provided": {
			input:         "http://abc.com?width=10",
			expectedError: `strconv.ParseUint: parsing "": invalid syntax`,
		},
		"Fails decoding if width not provided": {
			input:         "http://abc.com?height=10",
			expectedError: `strconv.ParseUint: parsing "": invalid syntax`,
		},
	}

	uriHelper := StrUriHelper{}
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)
		output, actualError := uriHelper.parseUri(test.input)

		if output != nil {
			t.Errorf("Expected return value nil, got %+v\n", output)
		}
		if actualError == nil {
			t.Errorf("Expected error not to be nil\n")
			break
		}
		if actualError.Error() != test.expectedError {
			t.Errorf("Expected error '%s', got '%s'\n", test.expectedError, actualError.Error())
		}
	}
}
