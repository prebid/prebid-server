package openrtb2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"

	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	gometrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
)

// From auction_test.go
// const maxSize = 1024 * 256

// TestGoodRequests makes sure that the auction runs properly-formatted stored bids correctly.
func TestGoodAmpRequests(t *testing.T) {
	goodRequests := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "aliased-buyeruids.json")),
		"2": json.RawMessage(validRequest(t, "aliases.json")),
		"5": json.RawMessage(validRequest(t, "gdpr-no-consentstring.json")),
		"6": json.RawMessage(validRequest(t, "gdpr.json")),
		"7": json.RawMessage(validRequest(t, "site.json")),
		"9": json.RawMessage(validRequest(t, "user.json")),
	}

	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	endpoint, _ := NewAmpEndpoint(
		&mockAmpExchange{},
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{goodRequests},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		newTestMetrics(),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
	)

	for requestID := range goodRequests {
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=%s", requestID), nil)
		recorder := httptest.NewRecorder()
		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status %d. Got %d. Request config ID was %s", http.StatusOK, recorder.Code, requestID)
			t.Errorf("Response body was: %s", recorder.Body)
			t.Errorf("Request was: %s", string(goodRequests[requestID]))
		}

		var response AmpResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Errorf("AMP response was: %s", recorder.Body.Bytes())
			t.Fatalf("Error unmarshalling response: %s", err.Error())
		}

		if response.Targeting == nil || len(response.Targeting) == 0 {
			t.Errorf("Bad response, no targeting data.\n Response was: %v", recorder.Body)
		}
		if len(response.Targeting) != 3 {
			t.Errorf("Bad targeting data. Expected 3 keys, got %d.", len(response.Targeting))
		}

		if response.Debug != nil {
			t.Errorf("Debug present but not requested")
		}

		assert.Equal(t, expectedErrorsFromHoldAuction, response.Errors, "errors")
	}
}

// Prevents #683
func TestAMPPageInfo(t *testing.T) {
	const page = "http://test.somepage.co.uk:1234?myquery=1&other=2"
	stored := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "site.json")),
	}
	exchange := &mockAmpExchange{}

	endpoint, _ := NewAmpEndpoint(
		exchange,
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{stored},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		newTestMetrics(),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
	)
	request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=1&curl=%s", url.QueryEscape(page)), nil)
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if !assert.NotNil(t, exchange.lastRequest, "Endpoint responded with %d: %s", recorder.Code, recorder.Body.String()) {
		return
	}
	if !assert.NotNil(t, exchange.lastRequest.Site) {
		return
	}
	assert.Equal(t, page, exchange.lastRequest.Site.Page)
	assert.Equal(t, "test.somepage.co.uk", exchange.lastRequest.Site.Domain)
}

func TestGDPRConsent(t *testing.T) {
	consent := "BOu5On0Ou5On0ADACHENAO7pqzAAppY"
	existingConsent := "BONV8oqONXwgmADACHENAO7pqzAAppY"

	testCases := []struct {
		description     string
		consent         string
		userExt         *openrtb_ext.ExtUser
		nilUser         bool
		expectedUserExt openrtb_ext.ExtUser
	}{
		{
			description: "Nil User",
			consent:     consent,
			nilUser:     true,
			expectedUserExt: openrtb_ext.ExtUser{
				Consent: consent,
			},
		},
		{
			description: "Nil User Ext",
			consent:     consent,
			userExt:     nil,
			expectedUserExt: openrtb_ext.ExtUser{
				Consent: consent,
			},
		},
		{
			description: "Overrides Existing Consent",
			consent:     consent,
			userExt: &openrtb_ext.ExtUser{
				Consent: existingConsent,
			},
			expectedUserExt: openrtb_ext.ExtUser{
				Consent: consent,
			},
		},
		{
			description: "Overrides Existing Consent - With Sibling Data",
			consent:     consent,
			userExt: &openrtb_ext.ExtUser{
				Consent: existingConsent,
			},
			expectedUserExt: openrtb_ext.ExtUser{
				Consent: consent,
			},
		},
		{
			description: "Does Not Override Existing Consent If Empty",
			consent:     "",
			userExt: &openrtb_ext.ExtUser{
				Consent: existingConsent,
			},
			expectedUserExt: openrtb_ext.ExtUser{
				Consent: existingConsent,
			},
		},
	}

	for _, test := range testCases {
		// Build Request
		bid, err := getTestBidRequest(test.nilUser, test.userExt, true, nil)
		if err != nil {
			t.Fatalf("Failed to marshal the complete openrtb2.BidRequest object %v", err)
		}

		// Simulated Stored Request Backend
		stored := map[string]json.RawMessage{"1": json.RawMessage(bid)}

		// Build Exchange Endpoint
		mockExchange := &mockAmpExchange{}
		endpoint, _ := NewAmpEndpoint(
			mockExchange,
			newParamsValidator(t),
			&mockAmpStoredReqFetcher{stored},
			empty_fetcher.EmptyFetcher{},
			&config.Configuration{MaxRequestSize: maxSize},
			newTestMetrics(),
			analyticsConf.NewPBSAnalytics(&config.Analytics{}),
			map[string]string{},
			[]byte{},
			openrtb_ext.BuildBidderMap(),
		)

		// Invoke Endpoint
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=1&consent_string=%s", test.consent), nil)
		responseRecorder := httptest.NewRecorder()
		endpoint(responseRecorder, request, nil)

		// Parse Response
		var response AmpResponse
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Error unmarshalling response: %s", err.Error())
		}

		// Assert Result
		result := mockExchange.lastRequest
		if !assert.NotNil(t, result, test.description+":lastRequest") {
			return
		}
		if !assert.NotNil(t, result.User, test.description+":lastRequest.User") {
			return
		}
		if !assert.NotNil(t, result.User.Ext, test.description+":lastRequest.User.Ext") {
			return
		}
		var ue openrtb_ext.ExtUser
		err = json.Unmarshal(result.User.Ext, &ue)
		if !assert.NoError(t, err, test.description+":deserialize") {
			return
		}
		assert.Equal(t, test.expectedUserExt, ue, test.description)
		assert.Equal(t, expectedErrorsFromHoldAuction, response.Errors, test.description+":errors")
		assert.Empty(t, response.Warnings, test.description+":warnings")

		// Invoke Endpoint With Legacy Param
		requestLegacy := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=1&gdpr_consent=%s", test.consent), nil)
		responseRecorderLegacy := httptest.NewRecorder()
		endpoint(responseRecorderLegacy, requestLegacy, nil)

		// Parse Resonse
		var responseLegacy AmpResponse
		if err := json.Unmarshal(responseRecorderLegacy.Body.Bytes(), &responseLegacy); err != nil {
			t.Fatalf("Error unmarshalling response: %s", err.Error())
		}

		// Assert Result With Legacy Param
		resultLegacy := mockExchange.lastRequest
		if !assert.NotNil(t, resultLegacy, test.description+":legacy:lastRequest") {
			return
		}
		if !assert.NotNil(t, resultLegacy.User, test.description+":legacy:lastRequest.User") {
			return
		}
		if !assert.NotNil(t, resultLegacy.User.Ext, test.description+":legacy:lastRequest.User.Ext") {
			return
		}
		var ueLegacy openrtb_ext.ExtUser
		err = json.Unmarshal(resultLegacy.User.Ext, &ueLegacy)
		if !assert.NoError(t, err, test.description+":legacy:deserialize") {
			return
		}
		assert.Equal(t, test.expectedUserExt, ueLegacy, test.description+":legacy")
		assert.Equal(t, expectedErrorsFromHoldAuction, responseLegacy.Errors, test.description+":legacy:errors")
		assert.Empty(t, responseLegacy.Warnings, test.description+":legacy:warnings")
	}
}

func TestCCPAConsent(t *testing.T) {
	consent := "1NYN"
	existingConsent := "1NNN"

	var gdpr int8 = 1

	testCases := []struct {
		description    string
		consent        string
		regsExt        *openrtb_ext.ExtRegs
		nilRegs        bool
		expectedRegExt openrtb_ext.ExtRegs
	}{
		{
			description: "Nil Regs",
			consent:     consent,
			nilRegs:     true,
			expectedRegExt: openrtb_ext.ExtRegs{
				USPrivacy: consent,
			},
		},
		{
			description: "Nil Regs Ext",
			consent:     consent,
			regsExt:     nil,
			expectedRegExt: openrtb_ext.ExtRegs{
				USPrivacy: consent,
			},
		},
		{
			description: "Overrides Existing Consent",
			consent:     consent,
			regsExt: &openrtb_ext.ExtRegs{
				USPrivacy: existingConsent,
			},
			expectedRegExt: openrtb_ext.ExtRegs{
				USPrivacy: consent,
			},
		},
		{
			description: "Overrides Existing Consent - With Sibling Data",
			consent:     consent,
			regsExt: &openrtb_ext.ExtRegs{
				USPrivacy: existingConsent,
				GDPR:      &gdpr,
			},
			expectedRegExt: openrtb_ext.ExtRegs{
				USPrivacy: consent,
				GDPR:      &gdpr,
			},
		},
		{
			description: "Does Not Override Existing Consent If Empty",
			consent:     "",
			regsExt: &openrtb_ext.ExtRegs{
				USPrivacy: existingConsent,
			},
			expectedRegExt: openrtb_ext.ExtRegs{
				USPrivacy: existingConsent,
			},
		},
	}

	for _, test := range testCases {
		// Build Request
		bid, err := getTestBidRequest(true, nil, test.nilRegs, test.regsExt)
		if err != nil {
			t.Fatalf("Failed to marshal the complete openrtb2.BidRequest object %v", err)
		}

		// Simulated Stored Request Backend
		stored := map[string]json.RawMessage{"1": json.RawMessage(bid)}

		// Build Exchange Endpoint
		mockExchange := &mockAmpExchange{}
		endpoint, _ := NewAmpEndpoint(
			mockExchange,
			newParamsValidator(t),
			&mockAmpStoredReqFetcher{stored},
			empty_fetcher.EmptyFetcher{},
			&config.Configuration{MaxRequestSize: maxSize},
			newTestMetrics(),
			analyticsConf.NewPBSAnalytics(&config.Analytics{}),
			map[string]string{},
			[]byte{},
			openrtb_ext.BuildBidderMap(),
		)

		// Invoke Endpoint
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=1&consent_string=%s", test.consent), nil)
		responseRecorder := httptest.NewRecorder()
		endpoint(responseRecorder, request, nil)

		// Parse Response
		var response AmpResponse
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Error unmarshalling response: %s", err.Error())
		}

		// Assert Result
		result := mockExchange.lastRequest
		if !assert.NotNil(t, result, test.description+":lastRequest") {
			return
		}
		if !assert.NotNil(t, result.Regs, test.description+":lastRequest.Regs") {
			return
		}
		if !assert.NotNil(t, result.Regs.Ext, test.description+":lastRequest.Regs.Ext") {
			return
		}
		var re openrtb_ext.ExtRegs
		err = json.Unmarshal(result.Regs.Ext, &re)
		if !assert.NoError(t, err, test.description+":deserialize") {
			return
		}
		assert.Equal(t, test.expectedRegExt, re, test.description)
		assert.Equal(t, expectedErrorsFromHoldAuction, response.Errors)
		assert.Empty(t, response.Warnings)
	}
}

func TestConsentWarnings(t *testing.T) {
	type inputTest struct {
		regs              *openrtb_ext.ExtRegs
		invalidConsentURL bool
		expectedWarnings  map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage
	}
	invalidConsent := "invalid"

	bidderWarning := openrtb_ext.ExtBidderMessage{
		Code:    10003,
		Message: "debug turned off for bidder",
	}
	invalidCCPAWarning := openrtb_ext.ExtBidderMessage{
		Code:    10001,
		Message: "Consent '" + invalidConsent + "' is not recognized as either CCPA or GDPR TCF.",
	}
	invalidConsentWarning := openrtb_ext.ExtBidderMessage{
		Code:    10001,
		Message: "CCPA consent is invalid and will be ignored. (request.regs.ext.us_privacy must contain 4 characters)",
	}

	testData := []inputTest{
		{
			regs:              nil,
			invalidConsentURL: false,
			expectedWarnings:  nil,
		},
		{
			regs:              nil,
			invalidConsentURL: true,
			expectedWarnings:  map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage{openrtb_ext.BidderReservedGeneral: {invalidCCPAWarning}},
		},
		{
			regs:              &openrtb_ext.ExtRegs{USPrivacy: "invalid"},
			invalidConsentURL: true,
			expectedWarnings: map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage{
				openrtb_ext.BidderReservedGeneral:  {invalidCCPAWarning, invalidConsentWarning},
				openrtb_ext.BidderName("appnexus"): {bidderWarning},
			},
		},
		{
			regs:              &openrtb_ext.ExtRegs{USPrivacy: "1NYN"},
			invalidConsentURL: false,
			expectedWarnings:  map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage{openrtb_ext.BidderName("appnexus"): {bidderWarning}},
		},
	}

	for _, testCase := range testData {

		bid, err := getTestBidRequest(true, nil, testCase.regs == nil, testCase.regs)
		if err != nil {
			t.Fatalf("Failed to marshal the complete openrtb2.BidRequest object %v", err)
		}

		// Simulated Stored Request Backend
		stored := map[string]json.RawMessage{"1": json.RawMessage(bid)}

		// Build Exchange Endpoint
		var mockExchange exchange.Exchange
		if testCase.regs != nil {
			mockExchange = &mockAmpExchangeWarnings{}
		} else {
			mockExchange = &mockAmpExchange{}
		}
		endpoint, _ := NewAmpEndpoint(
			mockExchange,
			newParamsValidator(t),
			&mockAmpStoredReqFetcher{stored},
			empty_fetcher.EmptyFetcher{},
			&config.Configuration{MaxRequestSize: maxSize},
			newTestMetrics(),
			analyticsConf.NewPBSAnalytics(&config.Analytics{}),
			map[string]string{},
			[]byte{},
			openrtb_ext.BuildBidderMap(),
		)

		// Invoke Endpoint
		var request *http.Request

		if testCase.invalidConsentURL {
			request = httptest.NewRequest("GET", "/openrtb2/auction/amp?tag_id=1&consent_string="+invalidConsent, nil)

		} else {
			request = httptest.NewRequest("GET", "/openrtb2/auction/amp?tag_id=1", nil)
		}

		responseRecorder := httptest.NewRecorder()
		endpoint(responseRecorder, request, nil)

		// Parse Response
		var response AmpResponse
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Error unmarshalling response: %s", err.Error())
		}

		// Assert Result
		if testCase.regs == nil {
			result := mockExchange.(*mockAmpExchange).lastRequest
			assert.NotNil(t, result, "lastRequest")
			assert.Nil(t, result.User, "lastRequest.User")
			assert.Nil(t, result.Regs, "lastRequest.Regs")
			assert.Equal(t, expectedErrorsFromHoldAuction, response.Errors)
			if testCase.invalidConsentURL {
				assert.Equal(t, testCase.expectedWarnings, response.Warnings)
			} else {
				assert.Empty(t, response.Warnings)
			}

		} else {
			assert.Equal(t, testCase.expectedWarnings, response.Warnings)
		}
	}
}

func TestNewAndLegacyConsentBothProvided(t *testing.T) {
	validConsentGDPR1 := "BOu5On0Ou5On0ADACHENAO7pqzAAppY"
	validConsentGDPR2 := "BONV8oqONXwgmADACHENAO7pqzAAppY"

	testCases := []struct {
		description     string
		consent         string
		consentLegacy   string
		userExt         *openrtb_ext.ExtUser
		expectedUserExt openrtb_ext.ExtUser
	}{
		{
			description:   "New Consent Wins",
			consent:       validConsentGDPR1,
			consentLegacy: validConsentGDPR2,
			expectedUserExt: openrtb_ext.ExtUser{
				Consent: validConsentGDPR1,
			},
		},
		{
			description:   "New Consent Wins - Reverse",
			consent:       validConsentGDPR2,
			consentLegacy: validConsentGDPR1,
			expectedUserExt: openrtb_ext.ExtUser{
				Consent: validConsentGDPR2,
			},
		},
	}

	for _, test := range testCases {
		// Build Request
		bid, err := getTestBidRequest(false, nil, true, nil)
		if err != nil {
			t.Fatalf("Failed to marshal the complete openrtb2.BidRequest object %v", err)
		}

		// Simulated Stored Request Backend
		stored := map[string]json.RawMessage{"1": json.RawMessage(bid)}

		// Build Exchange Endpoint
		mockExchange := &mockAmpExchange{}
		endpoint, _ := NewAmpEndpoint(
			mockExchange,
			newParamsValidator(t),
			&mockAmpStoredReqFetcher{stored},
			empty_fetcher.EmptyFetcher{},
			&config.Configuration{MaxRequestSize: maxSize},
			newTestMetrics(),
			analyticsConf.NewPBSAnalytics(&config.Analytics{}),
			map[string]string{},
			[]byte{},
			openrtb_ext.BuildBidderMap(),
		)

		// Invoke Endpoint
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=1&consent_string=%s&gdpr_consent=%s", test.consent, test.consentLegacy), nil)
		responseRecorder := httptest.NewRecorder()
		endpoint(responseRecorder, request, nil)

		// Parse Response
		var response AmpResponse
		if err := json.Unmarshal(responseRecorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Error unmarshalling response: %s", err.Error())
		}

		// Assert Result
		result := mockExchange.lastRequest
		if !assert.NotNil(t, result, test.description+":lastRequest") {
			return
		}
		if !assert.NotNil(t, result.User, test.description+":lastRequest.User") {
			return
		}
		if !assert.NotNil(t, result.User.Ext, test.description+":lastRequest.User.Ext") {
			return
		}
		var ue openrtb_ext.ExtUser
		err = json.Unmarshal(result.User.Ext, &ue)
		if !assert.NoError(t, err, test.description+":deserialize") {
			return
		}
		assert.Equal(t, test.expectedUserExt, ue, test.description)
		assert.Equal(t, expectedErrorsFromHoldAuction, response.Errors)
		assert.Empty(t, response.Warnings)
	}
}

func TestAMPSiteExt(t *testing.T) {
	stored := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "site.json")),
	}
	exchange := &mockAmpExchange{}
	endpoint, _ := NewAmpEndpoint(
		exchange,
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{stored},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		newTestMetrics(),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		nil,
		nil,
		openrtb_ext.BuildBidderMap(),
	)
	request, err := http.NewRequest("GET", "/openrtb2/auction/amp?tag_id=1", nil)
	if !assert.NoError(t, err) {
		return
	}
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if !assert.NotNil(t, exchange.lastRequest, "Endpoint responded with %d: %s", recorder.Code, recorder.Body.String()) {
		return
	}
	if !assert.NotNil(t, exchange.lastRequest.Site) {
		return
	}
	assert.JSONEq(t, `{"amp":1}`, string(exchange.lastRequest.Site.Ext))
}

// TestBadRequests makes sure we return 400's on bad requests.
func TestAmpBadRequests(t *testing.T) {
	files := fetchFiles(t, "sample-requests/invalid-whole")
	badRequests := make(map[string]json.RawMessage, len(files))
	for index, file := range files {
		badRequests[strconv.Itoa(100+index)] = readFile(t, "sample-requests/invalid-whole/"+file.Name())
	}

	endpoint, _ := NewAmpEndpoint(
		&mockAmpExchange{},
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{badRequests},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		newTestMetrics(),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
	)
	for requestID := range badRequests {
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=%s", requestID), nil)
		recorder := httptest.NewRecorder()

		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d. Got %d. Input was: %s", http.StatusBadRequest, recorder.Code, fmt.Sprintf("/openrtb2/auction/amp?config=%s", requestID))
		}
	}
}

// TestAmpDebug makes sure we get debug information back when requested
func TestAmpDebug(t *testing.T) {
	requests := map[string]json.RawMessage{
		"2": json.RawMessage(validRequest(t, "site.json")),
	}

	endpoint, _ := NewAmpEndpoint(
		&mockAmpExchange{},
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{requests},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		newTestMetrics(),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
	)

	for requestID := range requests {
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=%s&debug=1", requestID), nil)
		recorder := httptest.NewRecorder()
		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status %d. Got %d. Request config ID was %s", http.StatusOK, recorder.Code, requestID)
			t.Errorf("Response body was: %s", recorder.Body)
			t.Errorf("Request was: %s", string(requests[requestID]))
		}

		var response AmpResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Error unmarshalling response: %s", err.Error())
		}

		if response.Targeting == nil || len(response.Targeting) == 0 {
			t.Errorf("Bad response, no targeting data.\n Response was: %v", recorder.Body)
		}
		if len(response.Targeting) != 3 {
			t.Errorf("Bad targeting data. Expected 3 keys, got %d.", len(response.Targeting))
		}

		if response.Debug == nil {
			t.Errorf("Debug requested but not present")
		}
	}
}

// Prevents #452
func TestAmpTargetingDefaults(t *testing.T) {
	req := &openrtb2.BidRequest{}
	if errs := defaultRequestExt(req); len(errs) != 0 {
		t.Fatalf("Unexpected error defaulting request.ext for AMP: %v", errs)
	}

	var extRequest openrtb_ext.ExtRequest
	if err := json.Unmarshal(req.Ext, &extRequest); err != nil {
		t.Fatalf("Unexpected error unmarshalling defaulted request.ext for AMP: %v", err)
	}
	if extRequest.Prebid.Targeting == nil {
		t.Fatal("AMP defaults should set request.ext.targeting")
	}
	if !extRequest.Prebid.Targeting.IncludeWinners {
		t.Error("AMP defaults should set request.ext.targeting.includewinners to true")
	}
	if !extRequest.Prebid.Targeting.IncludeBidderKeys {
		t.Error("AMP defaults should set request.ext.targeting.includebidderkeys to true")
	}
	if !reflect.DeepEqual(extRequest.Prebid.Targeting.PriceGranularity, openrtb_ext.PriceGranularityFromString("med")) {
		t.Error("AMP defaults should set request.ext.targeting.pricegranularity to medium")
	}
}

func TestQueryParamOverrides(t *testing.T) {
	requests := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "site.json")),
	}

	endpoint, _ := NewAmpEndpoint(
		&mockAmpExchange{},
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{requests},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		newTestMetrics(),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
	)

	requestID := "1"
	curl := "http://example.com"
	slot := "1234"
	timeout := int64(500)
	account := "12345"

	request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=%s&debug=1&curl=%s&slot=%s&timeout=%d&account=%s", requestID, curl, slot, timeout, account), nil)
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d. Got %d. Request config ID was %s", http.StatusOK, recorder.Code, requestID)
		t.Errorf("Response body was: %s", recorder.Body)
		t.Errorf("Request was: %s", string(requests[requestID]))
	}

	var response AmpResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Error unmarshalling response: %s", err.Error())
	}

	resolvedRequest := response.Debug.ResolvedRequest
	if resolvedRequest.TMax != timeout {
		t.Errorf("Expected TMax to equal timeout (%d), got: %d", timeout, resolvedRequest.TMax)
	}

	resolvedImp := resolvedRequest.Imp[0]
	if resolvedImp.TagID != slot {
		t.Errorf("Expected Imp.TagId to equal slot (%s), got: %s", slot, resolvedImp.TagID)
	}

	if resolvedRequest.Site == nil || resolvedRequest.Site.Page != curl {
		t.Errorf("Expected Site.Page to equal curl (%s), got: %s", curl, resolvedRequest.Site.Page)
	}

	if resolvedRequest.Site == nil || resolvedRequest.Site.Publisher == nil || resolvedRequest.Site.Publisher.ID != account {
		t.Errorf("Expected Site.Publisher.ID to equal (%s), got: %s", account, resolvedRequest.Site.Publisher.ID)
	}
}

func TestOverrideDimensions(t *testing.T) {
	formatOverrideSpec{
		overrideWidth:  20,
		overrideHeight: 40,
		expect: []openrtb2.Format{{
			W: 20,
			H: 40,
		}},
	}.execute(t)
}

func TestOverrideHeightNormalWidth(t *testing.T) {
	formatOverrideSpec{
		width:          20,
		overrideHeight: 40,
		expect: []openrtb2.Format{{
			W: 20,
			H: 40,
		}},
	}.execute(t)
}

func TestOverrideWidthNormalHeight(t *testing.T) {
	formatOverrideSpec{
		overrideWidth: 20,
		height:        40,
		expect: []openrtb2.Format{{
			W: 20,
			H: 40,
		}},
	}.execute(t)
}

func TestMultisize(t *testing.T) {
	formatOverrideSpec{
		multisize: "200x50,100x60",
		expect: []openrtb2.Format{{
			W: 200,
			H: 50,
		}, {
			W: 100,
			H: 60,
		}},
	}.execute(t)
}

func TestSizeWithMultisize(t *testing.T) {
	formatOverrideSpec{
		width:     20,
		height:    40,
		multisize: "200x50,100x60",
		expect: []openrtb2.Format{{
			W: 20,
			H: 40,
		}, {
			W: 200,
			H: 50,
		}, {
			W: 100,
			H: 60,
		}},
	}.execute(t)
}

func TestHeightOnly(t *testing.T) {
	formatOverrideSpec{
		height: 200,
		expect: []openrtb2.Format{{
			W: 300,
			H: 200,
		}},
	}.execute(t)
}

func TestWidthOnly(t *testing.T) {
	formatOverrideSpec{
		width: 150,
		expect: []openrtb2.Format{{
			W: 150,
			H: 600,
		}},
	}.execute(t)
}

type formatOverrideSpec struct {
	width          uint64
	height         uint64
	overrideWidth  uint64
	overrideHeight uint64
	multisize      string
	account        string
	expect         []openrtb2.Format
}

func (s formatOverrideSpec) execute(t *testing.T) {
	requests := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "site.json")),
	}

	endpoint, _ := NewAmpEndpoint(
		&mockAmpExchange{},
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{requests},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		newTestMetrics(),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
	)

	url := fmt.Sprintf("/openrtb2/auction/amp?tag_id=1&debug=1&w=%d&h=%d&ow=%d&oh=%d&ms=%s&account=%s", s.width, s.height, s.overrideWidth, s.overrideHeight, s.multisize, s.account)
	request := httptest.NewRequest("GET", url, nil)
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d. Got %d. Request config ID was 1", http.StatusOK, recorder.Code)
		t.Errorf("Response body was: %s", recorder.Body)
		t.Errorf("Request was: %s", string(requests["1"]))
	}
	var response AmpResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Error unmarshalling response: %s", err.Error())
	}

	formats := response.Debug.ResolvedRequest.Imp[0].Banner.Format
	if len(formats) != len(s.expect) {
		t.Fatalf("Bad formats length. Expected %v, got %v", s.expect, formats)
	}
	for i := 0; i < len(formats); i++ {
		if formats[i].W != s.expect[i].W {
			t.Errorf("format[%d].W were not equal. Expected %d, got %d", i, s.expect[i].W, formats[i].W)
		}
		if formats[i].H != s.expect[i].H {
			t.Errorf("format[%d].H were not equal. Expected %d, got %d", i, s.expect[i].H, formats[i].H)
		}
	}
}

type mockAmpStoredReqFetcher struct {
	data map[string]json.RawMessage
}

func (cf *mockAmpStoredReqFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	return cf.data, nil, nil
}

type mockAmpExchange struct {
	lastRequest *openrtb2.BidRequest
}

var expectedErrorsFromHoldAuction map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage = map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage{
	openrtb_ext.BidderName("openx"): {
		{
			Code:    1,
			Message: "The request exceeded the timeout allocated",
		},
	},
}

func (m *mockAmpExchange) HoldAuction(ctx context.Context, r exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	m.lastRequest = r.BidRequest

	response := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{
			Bid: []openrtb2.Bid{{
				AdM: "<script></script>",
				Ext: json.RawMessage(`{ "prebid": {"targeting": { "hb_pb": "1.20", "hb_appnexus_pb": "1.20", "hb_cache_id": "some_id"}}}`),
			}},
		}},
		Ext: json.RawMessage(`{ "errors": {"openx":[ { "code": 1, "message": "The request exceeded the timeout allocated" } ] } }`),
	}

	if r.BidRequest.Test == 1 {
		resolvedRequest, err := json.Marshal(r.BidRequest)
		if err != nil {
			resolvedRequest = json.RawMessage("{}")
		}
		response.Ext = json.RawMessage(fmt.Sprintf(`{"debug": {"httpcalls": {}, "resolvedrequest": %s}}`, resolvedRequest))
	}

	return response, nil
}

type mockAmpExchangeWarnings struct{}

func (m *mockAmpExchangeWarnings) HoldAuction(ctx context.Context, r exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	response := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{
			Bid: []openrtb2.Bid{{
				AdM: "<script></script>",
				Ext: json.RawMessage(`{ "prebid": {"targeting": { "hb_pb": "1.20", "hb_appnexus_pb": "1.20", "hb_cache_id": "some_id"}}}`),
			}},
		}},
		Ext: json.RawMessage(`{ "warnings": {"appnexus": [{"code": 10003, "message": "debug turned off for bidder"}] }}`),
	}
	return response, nil
}

func getTestBidRequest(nilUser bool, userExt *openrtb_ext.ExtUser, nilRegs bool, regsExt *openrtb_ext.ExtRegs) ([]byte, error) {
	var width int64 = 300
	var height int64 = 300
	bidRequest := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID:  "/19968336/header-bid-tag-0",
				Ext: json.RawMessage(`{"appnexus": { "placementId":12883451 }}`),
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{
							W: width,
							H: 250,
						},
						{
							W: width,
							H: 240,
						},
					},
					W: &width,
					H: &height,
				},
			},
		},
		Site: &openrtb2.Site{
			ID:   "site-id",
			Page: "some-page",
		},
	}

	var userExtData []byte
	if userExt != nil {
		var err error
		userExtData, err = json.Marshal(userExt)
		if err != nil {
			return nil, err
		}
	}

	if !nilUser {
		bidRequest.User = &openrtb2.User{
			ID:       "aUserId",
			BuyerUID: "aBuyerID",
			Ext:      userExtData,
		}
	}

	var regsExtData []byte
	if regsExt != nil {
		var err error
		regsExtData, err = json.Marshal(regsExt)
		if err != nil {
			return nil, err
		}
	}

	if !nilRegs {
		bidRequest.Regs = &openrtb2.Regs{
			COPPA: 1,
			Ext:   regsExtData,
		}
	}
	return json.Marshal(bidRequest)
}

func TestSetEffectiveAmpPubID(t *testing.T) {
	testPubID := "test-pub"

	testCases := []struct {
		description   string
		req           *openrtb2.BidRequest
		account       string
		expectedPubID string
	}{
		{
			description: "No publisher ID provided",
			req: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: nil,
				},
			},
			expectedPubID: "",
		},
		{
			description: "Publisher ID present in req.App.Publisher.ID",
			req: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						ID: testPubID,
					},
				},
			},
			expectedPubID: testPubID,
		},
		{
			description: "Publisher ID present in req.Site.Publisher.ID",
			req: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						ID: testPubID,
					},
				},
			},
			expectedPubID: testPubID,
		},
		{
			description: "Publisher ID present in account parameter",
			req: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						ID: "",
					},
				},
			},
			account:       testPubID,
			expectedPubID: testPubID,
		},
		{
			description: "req.Site.Publisher present but ID set to empty string",
			req: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						ID: "",
					},
				},
			},
			expectedPubID: "",
		},
	}

	for _, test := range testCases {
		setEffectiveAmpPubID(test.req, test.account)
		if test.req.Site != nil {
			assert.Equal(t, test.expectedPubID, test.req.Site.Publisher.ID,
				"should return the expected Publisher ID for test case: %s", test.description)
		} else {
			assert.Equal(t, test.expectedPubID, test.req.App.Publisher.ID,
				"should return the expected Publisher ID for test case: %s", test.description)
		}
	}
}

type mockLogger struct {
	ampObject *analytics.AmpObject
}

func newMockLogger(ao *analytics.AmpObject) analytics.PBSAnalyticsModule {
	return &mockLogger{
		ampObject: ao,
	}
}

func (logger mockLogger) LogAuctionObject(ao *analytics.AuctionObject) {
	return
}
func (logger mockLogger) LogVideoObject(vo *analytics.VideoObject) {
	return
}
func (logger mockLogger) LogCookieSyncObject(cookieObject *analytics.CookieSyncObject) {
	return
}
func (logger mockLogger) LogSetUIDObject(uuidObj *analytics.SetUIDObject) {
	return
}
func (logger mockLogger) LogNotificationEventObject(uuidObj *analytics.NotificationEvent) {
	return
}
func (logger mockLogger) LogAmpObject(ao *analytics.AmpObject) {
	*logger.ampObject = *ao
}

func TestBuildAmpObject(t *testing.T) {
	testCases := []struct {
		description       string
		inTagId           string
		inStoredRequest   json.RawMessage
		expectedAmpObject *analytics.AmpObject
	}{
		{
			description:     "Stored Amp request with nil body. Only the error gets logged",
			inTagId:         "test",
			inStoredRequest: nil,
			expectedAmpObject: &analytics.AmpObject{
				Status: http.StatusOK,
				Errors: []error{fmt.Errorf("unexpected end of JSON input")},
			},
		},
		{
			description:     "Stored Amp request with no imps that should return error. Only the error gets logged",
			inTagId:         "test",
			inStoredRequest: json.RawMessage(`{"id":"some-request-id","site":{"page":"prebid.org"},"imp":[],"tmax":500}`),
			expectedAmpObject: &analytics.AmpObject{
				Status: http.StatusOK,
				Errors: []error{fmt.Errorf("data for tag_id='test' does not define the required imp array")},
			},
		},
		{
			description:     "Wrong tag_id, error gets logged",
			inTagId:         "unknown",
			inStoredRequest: json.RawMessage(`{"id":"some-request-id","site":{"page":"prebid.org"},"imp":[{"id":"some-impression-id","banner":{"format":[{"w":300,"h":250}]},"ext":{"appnexus":{"placementId":12883451}}}],"tmax":500}`),
			expectedAmpObject: &analytics.AmpObject{
				Status: http.StatusOK,
				Errors: []error{fmt.Errorf("unexpected end of JSON input")},
			},
		},
		{
			description:     "Valid stored Amp request, correct tag_id, a valid response should be logged",
			inTagId:         "test",
			inStoredRequest: json.RawMessage(`{"id":"some-request-id","site":{"page":"prebid.org"},"imp":[{"id":"some-impression-id","banner":{"format":[{"w":300,"h":250}]},"ext":{"appnexus":{"placementId":12883451}}}],"tmax":500}`),
			expectedAmpObject: &analytics.AmpObject{
				Status: http.StatusOK,
				Errors: nil,
				Request: &openrtb2.BidRequest{
					ID: "some-request-id",
					Device: &openrtb2.Device{
						IP: "192.0.2.1",
					},
					Site: &openrtb2.Site{
						Page:      "prebid.org",
						Publisher: &openrtb2.Publisher{},
						Ext:       json.RawMessage(`{"amp":1}`),
					},
					Imp: []openrtb2.Imp{
						{
							ID: "some-impression-id",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 300,
										H: 250,
									},
								},
							},
							Secure: func(val int8) *int8 { return &val }(1), //(*int8)(1),
							Ext:    json.RawMessage(`{"appnexus":{"placementId":12883451}}`),
						},
					},
					AT:   1,
					TMax: 500,
					Ext:  json.RawMessage(`{"prebid":{"cache":{"bids":{"returnCreative":null},"vastxml":null},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":20,"increment":0.1}]},"includewinners":true,"includebidderkeys":true,"includebrandcategory":null,"includeformat":false,"durationrangesec":null,"preferdeals":false}}}`),
				},
				AuctionResponse: &openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{{
						Bid: []openrtb2.Bid{{
							AdM: "<script></script>",
							Ext: json.RawMessage(`{ "prebid": {"targeting": { "hb_pb": "1.20", "hb_appnexus_pb": "1.20", "hb_cache_id": "some_id"}}}`),
						}},
						Seat: "",
					}},
					Ext: json.RawMessage(`{ "errors": {"openx":[ { "code": 1, "message": "The request exceeded the timeout allocated" } ] } }`),
				},
				AmpTargetingValues: map[string]string{
					"hb_appnexus_pb": "1.20",
					"hb_cache_id":    "some_id",
					"hb_pb":          "1.20",
				},
				Origin: "",
			},
		},
	}

	request := httptest.NewRequest("GET", "/openrtb2/auction/amp?tag_id=test", nil)
	recorder := httptest.NewRecorder()

	for _, test := range testCases {

		// Set up test, declare a new mock logger every time
		actualAmpObject := new(analytics.AmpObject)

		logger := newMockLogger(actualAmpObject)

		mockAmpFetcher := &mockAmpStoredReqFetcher{
			data: map[string]json.RawMessage{
				test.inTagId: json.RawMessage(test.inStoredRequest),
			},
		}

		endpoint, _ := NewAmpEndpoint(
			&mockAmpExchange{},
			newParamsValidator(t),
			mockAmpFetcher,
			empty_fetcher.EmptyFetcher{},
			&config.Configuration{MaxRequestSize: maxSize},
			newTestMetrics(),
			logger,
			map[string]string{},
			[]byte{},
			openrtb_ext.BuildBidderMap(),
		)

		// Run test
		endpoint(recorder, request, nil)

		// assert AmpObject
		assert.Equalf(t, test.expectedAmpObject.Status, actualAmpObject.Status, "Amp Object Status field doesn't match expected: %s\n", test.description)
		assert.Lenf(t, actualAmpObject.Errors, len(test.expectedAmpObject.Errors), "Amp Object Errors array doesn't match expected: %s\n", test.description)
		assert.Equalf(t, test.expectedAmpObject.Request, actualAmpObject.Request, "Amp Object BidRequest doesn't match expected: %s\n", test.description)
		assert.Equalf(t, test.expectedAmpObject.AuctionResponse, actualAmpObject.AuctionResponse, "Amp Object BidResponse doesn't match expected: %s\n", test.description)
		assert.Equalf(t, test.expectedAmpObject.AmpTargetingValues, actualAmpObject.AmpTargetingValues, "Amp Object AmpTargetingValues doesn't match expected: %s\n", test.description)
		assert.Equalf(t, test.expectedAmpObject.Origin, actualAmpObject.Origin, "Amp Object Origin field doesn't match expected: %s\n", test.description)
	}
}

func newTestMetrics() *metrics.Metrics {
	return metrics.NewMetrics(gometrics.NewRegistry(), openrtb_ext.CoreBidderNames(), config.DisabledMetrics{})
}
