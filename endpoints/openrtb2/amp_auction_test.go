package openrtb2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/amp"
	"github.com/prebid/prebid-server/analytics"
	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookexecution"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/metrics"
	metricsConfig "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
)

// TestGoodRequests makes sure that the auction runs properly-formatted stored bids correctly.
func TestGoodAmpRequests(t *testing.T) {
	testGroups := []struct {
		desc      string
		dir       string
		testFiles []string
	}{
		{
			desc: "Valid supplementary, tag_id param only",
			dir:  "sample-requests/amp/valid-supplementary/",
			testFiles: []string{
				"aliased-buyeruids.json",
				"aliases.json",
				"imp-with-stored-resp.json",
				"gdpr-no-consentstring.json",
				"gdpr.json",
			},
		},
		{
			desc: "Valid, consent handling in query",
			dir:  "sample-requests/amp/consent-through-query/",
			testFiles: []string{
				"addtl-consent-through-query.json",
				"gdpr-tcf1-consent-through-query.json",
				"gdpr-tcf2-consent-through-query.json",
				"gdpr-legacy-tcf2-consent-through-query.json",
				"gdpr-ccpa-through-query.json",
			},
		},
	}

	for _, tgroup := range testGroups {
		for _, filename := range tgroup.testFiles {
			// Read test case and unmarshal
			fileJsonData, err := os.ReadFile(tgroup.dir + filename)
			if !assert.NoError(t, err, "Failed to fetch a valid request: %v. Test file: %s", err, filename) {
				continue
			}

			test := testCase{}
			if !assert.NoError(t, json.Unmarshal(fileJsonData, &test), "Failed to unmarshal data from file: %s. Error: %v", filename, err) {
				continue
			}

			// build http request
			request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?%s", test.Query), nil)
			recorder := httptest.NewRecorder()

			// build the stored requests and configure endpoint conf
			query := request.URL.Query()
			tagID := query.Get("tag_id")
			if !assert.Greater(t, len(tagID), 0, "AMP test %s file is missing tag_id field", filename) {
				continue
			}

			test.storedRequest = map[string]json.RawMessage{tagID: test.BidRequest}
			test.endpointType = AMP_ENDPOINT

			cfg := &config.Configuration{
				MaxRequestSize: maxSize,
				GDPR:           config.GDPR{Enabled: true},
			}
			if test.Config != nil {
				cfg.BlacklistedApps = test.Config.BlacklistedApps
				cfg.BlacklistedAppMap = test.Config.getBlacklistedAppMap()
				cfg.BlacklistedAccts = test.Config.BlacklistedAccounts
				cfg.BlacklistedAcctMap = test.Config.getBlackListedAccountMap()
				cfg.AccountRequired = test.Config.AccountRequired
			}

			// Set test up
			ampEndpoint, ex, mockBidServers, mockCurrencyRatesServer, err := buildTestEndpoint(test, cfg)
			if !assert.NoError(t, err) {
				continue
			}

			// runTestCase
			ampEndpoint(recorder, request, nil)

			// Close servers
			for _, mockBidServer := range mockBidServers {
				mockBidServer.Close()
			}
			mockCurrencyRatesServer.Close()

			// Assertions
			if assert.Equal(t, test.ExpectedReturnCode, recorder.Code, "Expected status %d. Got %d. Amp test file: %s", http.StatusOK, recorder.Code, filename) {
				if test.ExpectedReturnCode == http.StatusOK {
					assert.JSONEq(t, string(test.ExpectedAmpResponse), string(recorder.Body.Bytes()), "Not the expected response. Test file: %s", filename)
				} else {
					assert.Equal(t, test.ExpectedErrorMessage, recorder.Body.String(), filename)
				}
			}
			if test.ExpectedValidatedBidReq != nil {
				// compare as json to ignore whitespace and ext field ordering
				actualJson, err := json.Marshal(ex.actualValidatedBidReq)
				if assert.NoError(t, err, "Error converting actual bid request to json. Test file: %s", filename) {
					assert.JSONEq(t, string(test.ExpectedValidatedBidReq), string(actualJson), "Not the expected validated request. Test file: %s", filename)
				}
			}
		}
	}
}

func TestAccountErrors(t *testing.T) {
	tests := []struct {
		description string
		storedReqID string
		filename    string
	}{
		{
			description: "Malformed account config",
			storedReqID: "1",
			filename:    "account-malformed/malformed-acct.json",
		},
		{
			description: "Blocked account",
			storedReqID: "1",
			filename:    "blacklisted/blacklisted-site-publisher.json",
		},
	}

	for _, tt := range tests {
		fileJsonData, err := os.ReadFile("sample-requests/" + tt.filename)
		if !assert.NoError(t, err, "Failed to fetch a valid request: %v. Test file: %s", err, tt.filename) {
			continue
		}

		test := testCase{}
		if !assert.NoError(t, json.Unmarshal(fileJsonData, &test), "Failed to unmarshal data from file: %s. Error: %v", tt.filename, err) {
			continue
		}
		test.storedRequest = map[string]json.RawMessage{tt.storedReqID: test.BidRequest}
		test.endpointType = AMP_ENDPOINT

		cfg := &config.Configuration{
			BlacklistedAccts:   []string{"bad_acct"},
			BlacklistedAcctMap: map[string]bool{"bad_acct": true},
			MaxRequestSize:     maxSize,
		}
		cfg.MarshalAccountDefaults()

		ampEndpoint, _, mockBidServers, mockCurrencyRatesServer, err := buildTestEndpoint(test, cfg)
		if !assert.NoError(t, err) {
			continue
		}

		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=%s&", tt.storedReqID), nil)
		recorder := httptest.NewRecorder()
		ampEndpoint(recorder, request, nil)

		for _, mockBidServer := range mockBidServers {
			mockBidServer.Close()
		}
		mockCurrencyRatesServer.Close()

		assert.Equal(t, test.ExpectedReturnCode, recorder.Code, "%s: %s", tt.description, tt.filename)
		assert.Equal(t, test.ExpectedErrorMessage, recorder.Body.String(), "%s: %s", tt.description, tt.filename)
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
		fakeUUIDGenerator{},
		exchange,
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{stored},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
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
	consent := "CPdiPIJPdiPIJACABBENAzCv_____3___wAAAQNd_X9cAAAAAAAA"
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
			fakeUUIDGenerator{},
			mockExchange,
			newParamsValidator(t),
			&mockAmpStoredReqFetcher{stored},
			empty_fetcher.EmptyFetcher{},
			&config.Configuration{
				MaxRequestSize: maxSize,
				GDPR:           config.GDPR{Enabled: true},
			},
			&metricsConfig.NilMetricsEngine{},
			analyticsConf.NewPBSAnalytics(&config.Analytics{}),
			map[string]string{},
			[]byte{},
			openrtb_ext.BuildBidderMap(),
			empty_fetcher.EmptyFetcher{},
			hooks.EmptyPlanBuilder{},
		)

		// Invoke Endpoint
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=1&consent_type=2&consent_string=%s", test.consent), nil)
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
		assert.Equal(t, expectedErrorsFromHoldAuction, response.ORTB2.Ext.Errors, test.description+":errors")
		assert.Empty(t, response.ORTB2.Ext.Warnings, test.description+":warnings")

		// Invoke Endpoint With Legacy Param
		requestLegacy := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=1&consent_type=2&gdpr_consent=%s", test.consent), nil)
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
		assert.Equal(t, expectedErrorsFromHoldAuction, responseLegacy.ORTB2.Ext.Errors, test.description+":legacy:errors")
		assert.Empty(t, responseLegacy.ORTB2.Ext.Warnings, test.description+":legacy:warnings")
	}
}

func TestOverrideWithParams(t *testing.T) {
	e := &endpointDeps{
		cfg: &config.Configuration{
			GDPR: config.GDPR{
				Enabled: true,
			},
		},
	}

	type testInput struct {
		ampParams  amp.Params
		bidRequest *openrtb2.BidRequest
	}
	type testOutput struct {
		bidRequest        *openrtb2.BidRequest
		errorMsgs         []string
		expectFatalErrors bool
	}
	testCases := []struct {
		desc     string
		given    testInput
		expected testOutput
	}{
		{
			desc: "bid request with no Site field - amp.Params empty - expect Site to be added",
			given: testInput{
				ampParams: amp.Params{},
				bidRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{{Banner: &openrtb2.Banner{Format: []openrtb2.Format{}}}},
				},
			},
			expected: testOutput{
				bidRequest: &openrtb2.BidRequest{
					Imp:  []openrtb2.Imp{{Banner: &openrtb2.Banner{Format: []openrtb2.Format{}}}},
					Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":1}`)},
				},
				errorMsgs: nil,
			},
		},
		{
			desc: "amp.Params with Size field - expect Site and Banner format fields to be added",
			given: testInput{
				ampParams: amp.Params{
					Size: amp.Size{
						Width:  480,
						Height: 320,
					},
				},
				bidRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{{Banner: &openrtb2.Banner{Format: []openrtb2.Format{}}}},
				},
			},
			expected: testOutput{
				bidRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 480,
										H: 320,
									},
								},
							},
						},
					},
					Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":1}`)},
				},
				errorMsgs: nil,
			},
		},
		{
			desc: "amp.Params with CanonicalURL field - expect Site to be aded with Page and Domain fields",
			given: testInput{
				ampParams:  amp.Params{CanonicalURL: "http://www.foobar.com"},
				bidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{Banner: &openrtb2.Banner{Format: []openrtb2.Format{}}}}},
			},
			expected: testOutput{
				bidRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{{Banner: &openrtb2.Banner{Format: []openrtb2.Format{}}}},
					Site: &openrtb2.Site{
						Page:   "http://www.foobar.com",
						Domain: "www.foobar.com",
						Ext:    json.RawMessage(`{"amp":1}`),
					},
				},
				errorMsgs: nil,
			},
		},
		{
			desc: "amp.Params with Trace field - expect ext.prebid.trace to be added",
			given: testInput{
				ampParams:  amp.Params{Trace: "verbose"},
				bidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{Banner: &openrtb2.Banner{Format: []openrtb2.Format{}}}}},
			},
			expected: testOutput{
				bidRequest: &openrtb2.BidRequest{
					Imp:  []openrtb2.Imp{{Banner: &openrtb2.Banner{Format: []openrtb2.Format{}}}},
					Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":1}`)},
					Ext:  json.RawMessage(`{"prebid":{"trace":"verbose"}}`),
				},
				errorMsgs: nil,
			},
		},
		{
			desc: "amp.Params with Trace field - expect ext.prebid.trace to be merged with existing ext fields",
			given: testInput{
				ampParams: amp.Params{Trace: "verbose"},
				bidRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{{Banner: &openrtb2.Banner{Format: []openrtb2.Format{}}}},
					Ext: json.RawMessage(`{"prebid":{"debug":true}}`),
				},
			},
			expected: testOutput{
				bidRequest: &openrtb2.BidRequest{
					Imp:  []openrtb2.Imp{{Banner: &openrtb2.Banner{Format: []openrtb2.Format{}}}},
					Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":1}`)},
					Ext:  json.RawMessage(`{"prebid":{"debug":true,"trace":"verbose"}}`),
				},
				errorMsgs: nil,
			},
		},
		{
			desc: "bid request with malformed User.Ext - amp.Params with AdditionalConsent - expect error",
			given: testInput{
				ampParams: amp.Params{AdditionalConsent: "1~X.X.X.X"},
				bidRequest: &openrtb2.BidRequest{
					Imp:  []openrtb2.Imp{{Banner: &openrtb2.Banner{Format: []openrtb2.Format{}}}},
					User: &openrtb2.User{Ext: json.RawMessage(`malformed`)},
				},
			},
			expected: testOutput{
				bidRequest: &openrtb2.BidRequest{
					Imp:  []openrtb2.Imp{{Banner: &openrtb2.Banner{Format: []openrtb2.Format{}}}},
					Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":1}`)},
					User: &openrtb2.User{Ext: json.RawMessage(`malformed`)},
				},
				errorMsgs:         []string{"invalid character 'm' looking for beginning of value"},
				expectFatalErrors: true,
			},
		},
		{
			desc: "bid request with valid imp[0].ext - amp.Params with malformed targeting value - expect error because imp[0].ext won't be unable to get merged with targeting values",
			given: testInput{
				ampParams: amp.Params{Targeting: "{123,}"},
				bidRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{
							Banner: &openrtb2.Banner{Format: []openrtb2.Format{}},
							Ext:    []byte(`{"appnexus":{"placementId":123}}`),
						},
					},
				},
			},
			expected: testOutput{
				bidRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{
							Banner: &openrtb2.Banner{Format: []openrtb2.Format{}},
							Ext:    json.RawMessage(`{"appnexus":{"placementId":123}}`),
						},
					},
					Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":1}`)},
				},
				errorMsgs: []string{"unable to merge imp.ext with targeting data, check targeting data is correct: Invalid JSON Patch"},
			},
		},
		{
			desc: "bid request with malformed user.ext.prebid - amp.Params with GDPR consent values - expect policy writer to return error",
			given: testInput{
				ampParams: amp.Params{
					ConsentType: amp.ConsentTCF2,
					Consent:     "CPdECS0PdECS0ACABBENAzCv_____3___wAAAQNd_X9cAAAAAAAA",
				},
				bidRequest: &openrtb2.BidRequest{
					Imp:  []openrtb2.Imp{{Banner: &openrtb2.Banner{Format: []openrtb2.Format{}}}},
					User: &openrtb2.User{Ext: json.RawMessage(`{"prebid":{malformed}}`)},
				},
			},
			expected: testOutput{
				bidRequest: &openrtb2.BidRequest{
					Imp:  []openrtb2.Imp{{Banner: &openrtb2.Banner{Format: []openrtb2.Format{}}}},
					User: &openrtb2.User{Ext: json.RawMessage(`{"prebid":{malformed}}`)},
					Site: &openrtb2.Site{Ext: json.RawMessage(`{"amp":1}`)},
				},
				errorMsgs:         []string{"invalid character 'm' looking for beginning of object key string"},
				expectFatalErrors: true,
			},
		},
	}

	for _, test := range testCases {
		errs := e.overrideWithParams(test.given.ampParams, test.given.bidRequest)

		assert.Equal(t, test.expected.bidRequest, test.given.bidRequest, test.desc)
		assert.Len(t, errs, len(test.expected.errorMsgs), test.desc)
		if len(test.expected.errorMsgs) > 0 {
			assert.Equal(t, test.expected.errorMsgs[0], errs[0].Error(), test.desc)
			assert.Equal(t, test.expected.expectFatalErrors, errortypes.ContainsFatalError(errs), test.desc)
		}
	}
}

func TestSetConsentedProviders(t *testing.T) {

	sampleBidRequest := &openrtb2.BidRequest{}

	testCases := []struct {
		description            string
		givenAdditionalConsent string
		givenBidRequest        *openrtb2.BidRequest
		expectedBidRequest     *openrtb2.BidRequest
		expectedError          bool
	}{
		{
			description:            "empty additional consent bid request unmodified",
			givenAdditionalConsent: "",
			givenBidRequest:        sampleBidRequest,
			expectedBidRequest:     sampleBidRequest,
			expectedError:          false,
		},
		{
			description:            "nil bid request, expect error",
			givenAdditionalConsent: "ADDITIONAL_CONSENT_STRING",
			givenBidRequest:        nil,
			expectedBidRequest:     nil,
			expectedError:          true,
		},
		{
			description:            "malformed user.ext, expect error",
			givenAdditionalConsent: "ADDITIONAL_CONSENT_STRING",
			givenBidRequest: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: json.RawMessage(`malformed`),
				},
			},
			expectedBidRequest: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: json.RawMessage(`malformed`),
				},
			},
			expectedError: true,
		},
		{
			description:            "non-empty additional consent bid request will carry this value in user.ext.ConsentedProvidersSettings.consented_providers",
			givenAdditionalConsent: "ADDITIONAL_CONSENT_STRING",
			givenBidRequest:        sampleBidRequest,
			expectedBidRequest: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: json.RawMessage(`{"ConsentedProvidersSettings":{"consented_providers":"ADDITIONAL_CONSENT_STRING"}}`),
				},
			},
			expectedError: false,
		},
	}

	for _, test := range testCases {
		err := setConsentedProviders(test.givenBidRequest, amp.Params{AdditionalConsent: test.givenAdditionalConsent})

		if test.expectedError {
			assert.Error(t, err, test.description)
		} else {
			assert.NoError(t, err, test.description)
		}
		assert.Equal(t, test.expectedBidRequest, test.givenBidRequest, test.description)
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
			fakeUUIDGenerator{},
			mockExchange,
			newParamsValidator(t),
			&mockAmpStoredReqFetcher{stored},
			empty_fetcher.EmptyFetcher{},
			&config.Configuration{MaxRequestSize: maxSize},
			&metricsConfig.NilMetricsEngine{},
			analyticsConf.NewPBSAnalytics(&config.Analytics{}),
			map[string]string{},
			[]byte{},
			openrtb_ext.BuildBidderMap(),
			empty_fetcher.EmptyFetcher{},
			hooks.EmptyPlanBuilder{},
		)

		// Invoke Endpoint
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=1&consent_type=3&consent_string=%s", test.consent), nil)
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
		assert.Equal(t, expectedErrorsFromHoldAuction, response.ORTB2.Ext.Errors)
		assert.Empty(t, response.ORTB2.Ext.Warnings)
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
		Message: "Consent string '" + invalidConsent + "' is not a valid CCPA consent string.",
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
			fakeUUIDGenerator{},
			mockExchange,
			newParamsValidator(t),
			&mockAmpStoredReqFetcher{stored},
			empty_fetcher.EmptyFetcher{},
			&config.Configuration{MaxRequestSize: maxSize},
			&metricsConfig.NilMetricsEngine{},
			analyticsConf.NewPBSAnalytics(&config.Analytics{}),
			map[string]string{},
			[]byte{},
			openrtb_ext.BuildBidderMap(),
			empty_fetcher.EmptyFetcher{},
			hooks.EmptyPlanBuilder{},
		)

		// Invoke Endpoint
		var request *http.Request

		if testCase.invalidConsentURL {
			request = httptest.NewRequest("GET", "/openrtb2/auction/amp?tag_id=1&consent_type=3&consent_string="+invalidConsent, nil)

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
			assert.Equal(t, expectedErrorsFromHoldAuction, response.ORTB2.Ext.Errors)
			if testCase.invalidConsentURL {
				assert.Equal(t, testCase.expectedWarnings, response.ORTB2.Ext.Warnings)
			} else {
				assert.Empty(t, response.ORTB2.Ext.Warnings)
			}

		} else {
			assert.Equal(t, testCase.expectedWarnings, response.ORTB2.Ext.Warnings)
		}
	}
}

func TestNewAndLegacyConsentBothProvided(t *testing.T) {
	validConsentGDPR1 := "COwGVJOOwGVJOADACHENAOCAAO6as_-AAAhoAFNLAAoAAAA"
	validConsentGDPR2 := "CPdiPIJPdiPIJACABBENAzCv_____3___wAAAQNd_X9cAAAAAAAA"

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
			fakeUUIDGenerator{},
			mockExchange,
			newParamsValidator(t),
			&mockAmpStoredReqFetcher{stored},
			empty_fetcher.EmptyFetcher{},
			&config.Configuration{
				MaxRequestSize: maxSize,
				GDPR:           config.GDPR{Enabled: true},
			},
			&metricsConfig.NilMetricsEngine{},
			analyticsConf.NewPBSAnalytics(&config.Analytics{}),
			map[string]string{},
			[]byte{},
			openrtb_ext.BuildBidderMap(),
			empty_fetcher.EmptyFetcher{},
			hooks.EmptyPlanBuilder{},
		)

		// Invoke Endpoint
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=1&consent_type=2&consent_string=%s&gdpr_consent=%s", test.consent, test.consentLegacy), nil)
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
		assert.Equal(t, expectedErrorsFromHoldAuction, response.ORTB2.Ext.Errors)
		assert.Empty(t, response.ORTB2.Ext.Warnings)
	}
}

func TestAMPSiteExt(t *testing.T) {
	stored := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "site.json")),
	}
	exchange := &mockAmpExchange{}
	endpoint, _ := NewAmpEndpoint(
		fakeUUIDGenerator{},
		exchange,
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{stored},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		nil,
		nil,
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
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
	dir := "sample-requests/invalid-whole"
	files, err := os.ReadDir(dir)
	assert.NoError(t, err, "Failed to read folder: %s", dir)

	badRequests := make(map[string]json.RawMessage, len(files))
	for index, file := range files {
		badRequests[strconv.Itoa(100+index)] = readFile(t, "sample-requests/invalid-whole/"+file.Name())
	}

	endpoint, _ := NewAmpEndpoint(
		fakeUUIDGenerator{},
		&mockAmpExchange{},
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{badRequests},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
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
		fakeUUIDGenerator{},
		&mockAmpExchange{},
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{requests},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
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

		if response.ORTB2.Ext.Debug == nil {
			t.Errorf("Debug requested but not present")
		}
	}
}

func TestInitAmpTargetingAndCache(t *testing.T) {
	trueVal := true
	emptyTargetingAndCache := &openrtb_ext.ExtRequestPrebid{
		Targeting: &openrtb_ext.ExtRequestTargeting{},
		Cache: &openrtb_ext.ExtRequestPrebidCache{
			Bids: &openrtb_ext.ExtRequestPrebidCacheBids{},
		},
	}

	testCases := []struct {
		name           string
		request        *openrtb2.BidRequest
		expectedPrebid *openrtb_ext.ExtRequestPrebid
		expectedErrs   []string
	}{
		{
			name:         "malformed",
			request:      &openrtb2.BidRequest{Ext: json.RawMessage("malformed")},
			expectedErrs: []string{"invalid character 'm' looking for beginning of value"},
		},
		{
			name:           "nil",
			request:        &openrtb2.BidRequest{},
			expectedPrebid: emptyTargetingAndCache,
		},
		{
			name:           "empty",
			request:        &openrtb2.BidRequest{Ext: json.RawMessage(`{"ext":{}}`)},
			expectedPrebid: emptyTargetingAndCache,
		},
		{
			name:           "missing targeting + cache",
			request:        &openrtb2.BidRequest{Ext: json.RawMessage(`{"ext":{"prebid":{}}}`)},
			expectedPrebid: emptyTargetingAndCache,
		},
		{
			name:    "missing targeting",
			request: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"cache":{"bids":{"returnCreative":true}}}}`)},
			expectedPrebid: &openrtb_ext.ExtRequestPrebid{
				Targeting: &openrtb_ext.ExtRequestTargeting{},
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids: &openrtb_ext.ExtRequestPrebidCacheBids{
						ReturnCreative: &trueVal,
					},
				},
			},
		},
		{
			name:    "missing cache",
			request: &openrtb2.BidRequest{Ext: json.RawMessage(`{"prebid":{"targeting":{"includewinners":true}}}`)},
			expectedPrebid: &openrtb_ext.ExtRequestPrebid{
				Targeting: &openrtb_ext.ExtRequestTargeting{
					IncludeWinners: &trueVal,
				},
				Cache: &openrtb_ext.ExtRequestPrebidCache{
					Bids: &openrtb_ext.ExtRequestPrebidCacheBids{},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup
			req := &openrtb_ext.RequestWrapper{BidRequest: tc.request}

			// run
			actualErrs := initAmpTargetingAndCache(req)

			// assertions
			require.NoError(t, req.RebuildRequest(), "rebuild request")

			actualErrsMsgs := make([]string, len(actualErrs))
			for i, v := range actualErrs {
				actualErrsMsgs[i] = v.Error()
			}
			assert.ElementsMatch(t, tc.expectedErrs, actualErrsMsgs, "errors")

			actualReqExt, _ := req.GetRequestExt()
			actualPrebid := actualReqExt.GetPrebid()
			assert.Equal(t, tc.expectedPrebid, actualPrebid, "prebid ext")
		})
	}
}

func TestQueryParamOverrides(t *testing.T) {
	requests := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "site.json")),
	}

	endpoint, _ := NewAmpEndpoint(
		fakeUUIDGenerator{},
		&mockAmpExchange{},
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{requests},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
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

	var resolvedRequest openrtb2.BidRequest
	err := json.Unmarshal(response.ORTB2.Ext.Debug.ResolvedRequest, &resolvedRequest)
	assert.NoError(t, err, "resolved request should have a correct format")
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
		fakeUUIDGenerator{},
		&mockAmpExchange{},
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{requests},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
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
	var resolvedRequest openrtb2.BidRequest
	err := json.Unmarshal(response.ORTB2.Ext.Debug.ResolvedRequest, &resolvedRequest)
	assert.NoError(t, err, "resolved request should have the correct format")
	formats := resolvedRequest.Imp[0].Banner.Format
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

type mockAmpExchange struct {
	lastRequest *openrtb2.BidRequest
	requestExt  json.RawMessage
}

var expectedErrorsFromHoldAuction map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage = map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage{
	openrtb_ext.BidderName("openx"): {
		{
			Code:    1,
			Message: "The request exceeded the timeout allocated",
		},
	},
}

func (m *mockAmpExchange) HoldAuction(ctx context.Context, auctionRequest exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	r := auctionRequest.BidRequestWrapper
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

	if m.requestExt != nil {
		response.Ext = m.requestExt
	}
	if len(auctionRequest.StoredAuctionResponses) > 0 {
		var seatBids []openrtb2.SeatBid

		if err := json.Unmarshal(auctionRequest.StoredAuctionResponses[r.BidRequest.Imp[0].ID], &seatBids); err != nil {
			return nil, err
		}
		response.SeatBid = seatBids
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
			if test.req.Site.Publisher == nil {
				assert.Empty(t, test.expectedPubID,
					"should return the expected Publisher ID for test case: %s", test.description)
			} else {
				assert.Equal(t, test.expectedPubID, test.req.Site.Publisher.ID,
					"should return the expected Publisher ID for test case: %s", test.description)
			}
		} else {
			if test.req.App.Publisher == nil {
				assert.Empty(t, test.expectedPubID,
					"should return the expected Publisher ID for test case: %s", test.description)
			} else {
				assert.Equal(t, test.expectedPubID, test.req.App.Publisher.ID,
					"should return the expected Publisher ID for test case: %s", test.description)
			}
		}
	}
}

type mockLogger struct {
	ampObject     *analytics.AmpObject
	auctionObject *analytics.AuctionObject
}

func newMockLogger(ao *analytics.AmpObject, aucObj *analytics.AuctionObject) analytics.PBSAnalyticsModule {
	return &mockLogger{
		ampObject:     ao,
		auctionObject: aucObj,
	}
}

func (logger mockLogger) LogAuctionObject(ao *analytics.AuctionObject) {
	*logger.auctionObject = *ao
}
func (logger mockLogger) LogVideoObject(vo *analytics.VideoObject) {
}
func (logger mockLogger) LogCookieSyncObject(cookieObject *analytics.CookieSyncObject) {
}
func (logger mockLogger) LogSetUIDObject(uuidObj *analytics.SetUIDObject) {
}
func (logger mockLogger) LogNotificationEventObject(uuidObj *analytics.NotificationEvent) {
}
func (logger mockLogger) LogAmpObject(ao *analytics.AmpObject) {
	*logger.ampObject = *ao
}

func TestBuildAmpObject(t *testing.T) {
	testCases := []struct {
		description       string
		inTagId           string
		exchange          *mockAmpExchange
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
			inStoredRequest: json.RawMessage(`{"id":"some-request-id","site":{"page":"prebid.org"},"imp":[{"id":"some-impression-id","banner":{"format":[{"w":300,"h":250}]},"ext":{"prebid":{"bidder":{"appnexus":{"placementId":12883451}}}}}],"tmax":500}`),
			expectedAmpObject: &analytics.AmpObject{
				Status: http.StatusOK,
				Errors: []error{fmt.Errorf("unexpected end of JSON input")},
			},
		},
		{
			description:     "Valid stored Amp request, correct tag_id, a valid response should be logged",
			inTagId:         "test",
			inStoredRequest: json.RawMessage(`{"id":"some-request-id","site":{"page":"prebid.org"},"imp":[{"id":"some-impression-id","banner":{"format":[{"w":300,"h":250}]},"ext":{"prebid":{"bidder":{"appnexus":{"placementId":12883451}}}}}],"tmax":500}`),
			expectedAmpObject: &analytics.AmpObject{
				Status: http.StatusOK,
				Errors: nil,
				Request: &openrtb2.BidRequest{
					ID: "some-request-id",
					Device: &openrtb2.Device{
						IP: "192.0.2.1",
					},
					Site: &openrtb2.Site{
						Page: "prebid.org",
						Ext:  json.RawMessage(`{"amp":1}`),
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
							Ext:    json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placementId":12883451}}}}`),
						},
					},
					AT:   1,
					TMax: 500,
					Ext:  json.RawMessage(`{"prebid":{"cache":{"bids":{}},"channel":{"name":"amp","version":""},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":20,"increment":0.1}]},"includewinners":true,"includebidderkeys":true}}}`),
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
		{
			description:     "Global targeting from bid response should be applied for Amp",
			inTagId:         "test",
			inStoredRequest: json.RawMessage(`{"id":"some-request-id","site":{"page":"prebid.org"},"imp":[{"id":"some-impression-id","banner":{"format":[{"w":300,"h":250}]},"ext":{"prebid":{"bidder":{"appnexus":{"placementId":12883451}}}}}],"tmax":500}`),
			exchange:        &mockAmpExchange{requestExt: json.RawMessage(`{ "prebid": {"targeting": { "test_key": "test_value", "hb_appnexus_pb": "9999" } }, "errors": {"openx":[ { "code": 1, "message": "The request exceeded the timeout allocated" } ] } }`)},
			expectedAmpObject: &analytics.AmpObject{
				Status: http.StatusOK,
				Errors: nil,
				Request: &openrtb2.BidRequest{
					ID: "some-request-id",
					Device: &openrtb2.Device{
						IP: "192.0.2.1",
					},
					Site: &openrtb2.Site{
						Page: "prebid.org",
						Ext:  json.RawMessage(`{"amp":1}`),
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
							Ext:    json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placementId":12883451}}}}`),
						},
					},
					AT:   1,
					TMax: 500,
					Ext:  json.RawMessage(`{"prebid":{"cache":{"bids":{}},"channel":{"name":"amp","version":""},"targeting":{"pricegranularity":{"precision":2,"ranges":[{"min":0,"max":20,"increment":0.1}]},"includewinners":true,"includebidderkeys":true}}}`),
				},
				AuctionResponse: &openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{{
						Bid: []openrtb2.Bid{{
							AdM: "<script></script>",
							Ext: json.RawMessage(`{ "prebid": {"targeting": { "hb_pb": "1.20", "hb_appnexus_pb": "1.20", "hb_cache_id": "some_id"}}}`),
						}},
						Seat: "",
					}},
					Ext: json.RawMessage(`{ "prebid": {"targeting": { "test_key": "test_value", "hb_appnexus_pb": "9999" } }, "errors": {"openx":[ { "code": 1, "message": "The request exceeded the timeout allocated" } ] } }`),
				},
				AmpTargetingValues: map[string]string{
					"hb_appnexus_pb": "1.20", // Bid level has higher priority than global
					"hb_cache_id":    "some_id",
					"hb_pb":          "1.20",
					"test_key":       "test_value", // New global key added
				},
				Origin: "",
			},
		},
	}

	request := httptest.NewRequest("GET", "/openrtb2/auction/amp?tag_id=test", nil)
	recorder := httptest.NewRecorder()

	for _, test := range testCases {
		// Set up test, declare a new mock logger every time
		exchange := test.exchange
		if exchange == nil {
			exchange = &mockAmpExchange{}
		}
		actualAmpObject, endpoint := ampObjectTestSetup(t, test.inTagId, test.inStoredRequest, false, exchange)
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

func TestIdGeneration(t *testing.T) {
	uuid := "foo"

	testCases := []struct {
		description            string
		givenInStoredRequest   json.RawMessage
		givenGenerateRequestID bool
		expectedID             string
	}{
		{
			description:            "The givenGenerateRequestID flag is set to true, so even though the stored amp request already has an id, we should still generate a new uuid",
			givenInStoredRequest:   json.RawMessage(`{"id":"ThisID","site":{"page":"prebid.org"},"imp":[{"id":"some-imp-id","banner":{"format":[{"w":300,"h":250}]},"ext":{"appnexus":{"placementId":1}}}],"tmax":1}`),
			givenGenerateRequestID: true,
			expectedID:             uuid,
		},
		{
			description:            "The givenGenerateRequestID flag is set to true and the stored amp request ID is blank, so we should generate a new uuid for the request",
			givenInStoredRequest:   json.RawMessage(`{"id":"","site":{"page":"prebid.org"},"imp":[{"id":"some-imp-id","banner":{"format":[{"w":300,"h":250}]},"ext":{"appnexus":{"placementId":1}}}],"tmax":1}`),
			givenGenerateRequestID: true,
			expectedID:             uuid,
		},
		{
			description:            "The givenGenerateRequestID flag is false, so the ID shouldn't change",
			givenInStoredRequest:   json.RawMessage(`{"id":"ThisID","site":{"page":"prebid.org"},"imp":[{"id":"some-imp-id","banner":{"format":[{"w":300,"h":250}]},"ext":{"appnexus":{"placementId":1}}}],"tmax":1}`),
			givenGenerateRequestID: false,
			expectedID:             "ThisID",
		},
		{
			description:            "The givenGenerateRequestID flag is true, and the id field isn't included in the stored request, we should still generate a uuid",
			givenInStoredRequest:   json.RawMessage(`{"site":{"page":"prebid.org"},"imp":[{"id":"some-imp-id","banner":{"format":[{"w":300,"h":250}]},"ext":{"appnexus":{"placementId":1}}}],"tmax":1}`),
			givenGenerateRequestID: true,
			expectedID:             uuid,
		},
		{
			description:            "The givenGenerateRequestID flag is false, but id field is the macro option {{UUID}}, we should generate a uuid",
			givenInStoredRequest:   json.RawMessage(`{"id":"{{UUID}}","site":{"page":"prebid.org"},"imp":[{"id":"some-imp-id","banner":{"format":[{"w":300,"h":250}]},"ext":{"appnexus":{"placementId":1}}}],"tmax":1}`),
			givenGenerateRequestID: false,
			expectedID:             uuid,
		},
		{
			description:            "Macro ID case sensitivity check. The id is {{uuid}}, but we should only generate an id if it's all uppercase {{UUID}}. So the ID shouldn't change.",
			givenInStoredRequest:   json.RawMessage(`{"id":"{{uuid}}","site":{"page":"prebid.org"},"imp":[{"id":"some-imp-id","banner":{"format":[{"w":300,"h":250}]},"ext":{"appnexus":{"placementId":1}}}],"tmax":1}`),
			givenGenerateRequestID: false,
			expectedID:             "{{uuid}}",
		},
	}

	request := httptest.NewRequest("GET", "/openrtb2/auction/amp?tag_id=test", nil)
	recorder := httptest.NewRecorder()

	for _, test := range testCases {
		// Set up and run test
		actualAmpObject, endpoint := ampObjectTestSetup(t, "test", test.givenInStoredRequest, test.givenGenerateRequestID, &mockAmpExchange{})
		endpoint(recorder, request, nil)
		assert.Equalf(t, test.expectedID, actualAmpObject.Request.ID, "Bid Request ID is incorrect: %s\n", test.description)
	}
}

func ampObjectTestSetup(t *testing.T, inTagId string, inStoredRequest json.RawMessage, generateRequestID bool, exchange *mockAmpExchange) (*analytics.AmpObject, httprouter.Handle) {
	actualAmpObject := analytics.AmpObject{}
	logger := newMockLogger(&actualAmpObject, nil)

	mockAmpFetcher := &mockAmpStoredReqFetcher{
		data: map[string]json.RawMessage{
			inTagId: json.RawMessage(inStoredRequest),
		},
	}

	endpoint, _ := NewAmpEndpoint(
		fakeUUIDGenerator{id: "foo", err: nil},
		exchange,
		newParamsValidator(t),
		mockAmpFetcher,
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize, GenerateRequestID: generateRequestID},
		&metricsConfig.NilMetricsEngine{},
		logger,
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
	)
	return &actualAmpObject, endpoint
}

func TestAmpAuctionResponseHeaders(t *testing.T) {
	testCases := []struct {
		description         string
		requestURLArguments string
		expectedStatus      int
		expectedHeaders     func(http.Header)
	}{
		{
			description:         "Success Response",
			requestURLArguments: "?tag_id=1&__amp_source_origin=foo",
			expectedStatus:      200,
			expectedHeaders: func(h http.Header) {
				h.Set("AMP-Access-Control-Allow-Source-Origin", "foo")
				h.Set("Access-Control-Expose-Headers", "AMP-Access-Control-Allow-Source-Origin")
				h.Set("X-Prebid", "pbs-go/unknown")
				h.Set("Content-Type", "text/plain; charset=utf-8")
			},
		},
		{
			description:         "Failure Response",
			requestURLArguments: "?tag_id=invalid&__amp_source_origin=foo",
			expectedStatus:      400,
			expectedHeaders: func(h http.Header) {
				h.Set("AMP-Access-Control-Allow-Source-Origin", "foo")
				h.Set("Access-Control-Expose-Headers", "AMP-Access-Control-Allow-Source-Origin")
				h.Set("X-Prebid", "pbs-go/unknown")
			},
		},
	}

	storedRequests := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "site.json")),
	}
	exchange := &nobidExchange{}
	endpoint, _ := NewAmpEndpoint(
		fakeUUIDGenerator{},
		exchange,
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{storedRequests},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
	)

	for _, test := range testCases {
		httpReq := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp"+test.requestURLArguments), nil)
		recorder := httptest.NewRecorder()

		endpoint(recorder, httpReq, nil)

		expectedHeaders := http.Header{}
		test.expectedHeaders(expectedHeaders)

		assert.Equal(t, test.expectedStatus, recorder.Result().StatusCode, test.description+":statuscode")
		assert.Equal(t, expectedHeaders, recorder.Result().Header, test.description+":statuscode")
	}
}

func TestRequestWithTargeting(t *testing.T) {
	stored := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "site.json")),
	}
	exchange := &mockAmpExchange{}
	endpoint, _ := NewAmpEndpoint(
		fakeUUIDGenerator{},
		exchange,
		newParamsValidator(t),
		&mockAmpStoredReqFetcher{stored},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		nil,
		nil,
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
	)
	url, err := url.Parse("/openrtb2/auction/amp")
	assert.NoError(t, err, "unexpected error received while parsing url")
	values := url.Query()
	values.Add("targeting", `{"gam-key1":"val1", "gam-key2":"val2"}`)
	values.Add("tag_id", "1")
	url.RawQuery = values.Encode()

	request, err := http.NewRequest("GET", url.String(), nil)
	if !assert.NoError(t, err) {
		return
	}
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if assert.NotNil(t, exchange.lastRequest, "Endpoint responded with %d: %s", recorder.Code, recorder.Body.String()) {
		assert.JSONEq(t, `{"prebid":{"bidder":{"appnexus":{"placementId":12883451}}}, "data":{"gam-key1":"val1", "gam-key2":"val2"}}`, string(exchange.lastRequest.Imp[0].Ext))
	}
}

func TestSetTargeting(t *testing.T) {
	tests := []struct {
		description    string
		bidRequest     openrtb2.BidRequest
		targeting      string
		expectedImpExt string
		wantError      bool
		errorMessage   string
	}{
		{
			description:    "valid imp ext, valid targeting data",
			bidRequest:     openrtb2.BidRequest{Imp: []openrtb2.Imp{{Ext: []byte(`{"appnexus":{"placementId":123}}`)}}},
			targeting:      `{"gam-key1":"val1", "gam-key2":"val2"}`,
			expectedImpExt: `{"appnexus":{"placementId":123}, "data": {"gam-key1":"val1", "gam-key2":"val2"}}`,
			wantError:      false,
			errorMessage:   "",
		},
		{
			description:    "valid imp ext, empty targeting data",
			bidRequest:     openrtb2.BidRequest{Imp: []openrtb2.Imp{{Ext: []byte(`{"appnexus":{"placementId":123}}`)}}},
			targeting:      ``,
			expectedImpExt: `{"appnexus":{"placementId":123}}`,
			wantError:      false,
			errorMessage:   "",
		},
		{
			description:    "empty imp ext, valid targeting data",
			bidRequest:     openrtb2.BidRequest{Imp: []openrtb2.Imp{{Ext: []byte(`{}`)}}},
			targeting:      `{"gam-key1":"val1", "gam-key2":"val2"}`,
			expectedImpExt: `{"data": {"gam-key1":"val1", "gam-key2":"val2"}}`,
			wantError:      false,
			errorMessage:   "",
		},
		{
			description:    "nil imp ext, valid targeting data",
			bidRequest:     openrtb2.BidRequest{Imp: []openrtb2.Imp{{Ext: nil}}},
			targeting:      `{"gam-key1":"val1", "gam-key2":"val2"}`,
			expectedImpExt: `{"data": {"gam-key1":"val1", "gam-key2":"val2"}}`,
			wantError:      false,
			errorMessage:   "",
		},
		{
			description:    "imp ext has data, valid targeting data",
			bidRequest:     openrtb2.BidRequest{Imp: []openrtb2.Imp{{Ext: []byte(`{"data":{"placementId":123}}`)}}},
			targeting:      `{"gam-key1":"val1", "gam-key2":"val2"}`,
			expectedImpExt: `{"data": {"gam-key1":"val1", "gam-key2":"val2", "placementId":123}}`,
			wantError:      false,
			errorMessage:   "",
		},
		{
			description:    "imp ext has data and other fields, valid targeting data",
			bidRequest:     openrtb2.BidRequest{Imp: []openrtb2.Imp{{Ext: []byte(`{"data":{"placementId":123}, "prebid": 123}`)}}},
			targeting:      `{"gam-key1":"val1", "gam-key2":"val2"}`,
			expectedImpExt: `{"data": {"gam-key1":"val1", "gam-key2":"val2", "placementId":123}, "prebid":123}`,
			wantError:      false,
			errorMessage:   "",
		},
		{
			description:    "imp ext has invalid format, valid targeting data",
			bidRequest:     openrtb2.BidRequest{Imp: []openrtb2.Imp{{Ext: []byte(`{123:{}`)}}},
			targeting:      `{"gam-key1":"val1", "gam-key2":"val2"}`,
			expectedImpExt: ``,
			wantError:      true,
			errorMessage:   "unable to merge imp.ext with targeting data, check targeting data is correct: Invalid JSON Document",
		},
		{
			description:    "valid imp ext, invalid targeting data",
			bidRequest:     openrtb2.BidRequest{Imp: []openrtb2.Imp{{Ext: []byte(`{"appnexus":{"placementId":123}}`)}}},
			targeting:      `{123,}`,
			expectedImpExt: ``,
			wantError:      true,
			errorMessage:   "unable to merge imp.ext with targeting data, check targeting data is correct: Invalid JSON Patch",
		},
	}

	for _, test := range tests {
		req := &test.bidRequest
		err := setTargeting(req, test.targeting)
		if test.wantError {
			assert.EqualErrorf(t, err, test.errorMessage, "error is incorrect for test case: %s", test.description)
		} else {
			assert.NoError(t, err, "error should be nil for test case: %s", test.description)
			assert.JSONEq(t, test.expectedImpExt, string(req.Imp[0].Ext), "incorrect impression extension returned for test %s", test.description)
		}

	}
}

func TestValidAmpResponseWhenRequestRejected(t *testing.T) {
	const nbr int = 123

	testCases := []struct {
		description string
		file        string
		planBuilder hooks.ExecutionPlanBuilder
	}{
		{
			description: "Assert correct AmpResponse when request rejected at entrypoint stage",
			file:        "sample-requests/hooks/amp_entrypoint_reject.json",
			planBuilder: mockPlanBuilder{entrypointPlan: makePlan[hookstage.Entrypoint](mockRejectionHook{nbr})},
		},
		{
			// raw_auction stage not executed for AMP endpoint, so we expect full response
			description: "Assert correct AmpResponse when request rejected at raw_auction stage",
			file:        "sample-requests/amp/valid-supplementary/aliased-buyeruids.json",
			planBuilder: mockPlanBuilder{rawAuctionPlan: makePlan[hookstage.RawAuctionRequest](mockRejectionHook{nbr})},
		},
		{
			description: "Assert correct AmpResponse when request rejected at processed_auction stage",
			file:        "sample-requests/hooks/amp_processed_auction_request_reject.json",
			planBuilder: mockPlanBuilder{processedAuctionPlan: makePlan[hookstage.ProcessedAuctionRequest](mockRejectionHook{nbr})},
		},
		{
			// bidder_request stage rejects only bidder, so we expect bidder rejection warning added
			description: "Assert correct AmpResponse when request rejected at bidder-request stage",
			file:        "sample-requests/hooks/amp_bidder_reject.json",
			planBuilder: mockPlanBuilder{bidderRequestPlan: makePlan[hookstage.BidderRequest](mockRejectionHook{nbr})},
		},
		{
			// raw_bidder_response stage rejects only bidder, so we expect bidder rejection warning added
			description: "Assert correct AmpResponse when request rejected at raw_bidder_response stage",
			file:        "sample-requests/hooks/amp_bidder_response_reject.json",
			planBuilder: mockPlanBuilder{rawBidderResponsePlan: makePlan[hookstage.RawBidderResponse](mockRejectionHook{nbr})},
		},
		{
			// no debug information should be added for raw_auction stage because it's not executed for amp endpoint
			description: "Assert correct AmpResponse with debug information from modules added to ext.prebid.modules",
			file:        "sample-requests/hooks/amp.json",
			planBuilder: mockPlanBuilder{
				entrypointPlan: hooks.Plan[hookstage.Entrypoint]{
					{
						Timeout: 5 * time.Millisecond,
						Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
							entryPointHookUpdateWithErrors,
							entryPointHookUpdateWithErrorsAndWarnings,
						},
					},
					{
						Timeout: 5 * time.Millisecond,
						Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
							entryPointHookUpdate,
						},
					},
				},
				rawAuctionPlan: hooks.Plan[hookstage.RawAuctionRequest]{
					{
						Timeout: 5 * time.Millisecond,
						Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
							rawAuctionHookNone,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fileData, err := os.ReadFile(tc.file)
			assert.NoError(t, err, "Failed to read test file.")

			test := testCase{}
			assert.NoError(t, json.Unmarshal(fileData, &test), "Failed to parse test file.")

			request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?%s", test.Query), nil)
			recorder := httptest.NewRecorder()
			query := request.URL.Query()
			tagID := query.Get("tag_id")

			test.storedRequest = map[string]json.RawMessage{tagID: test.BidRequest}
			test.planBuilder = tc.planBuilder
			test.endpointType = AMP_ENDPOINT

			cfg := &config.Configuration{MaxRequestSize: maxSize, AccountDefaults: config.Account{DebugAllow: true}}
			ampEndpointHandler, _, mockBidServers, mockCurrencyRatesServer, err := buildTestEndpoint(test, cfg)
			assert.NoError(t, err, "Failed to build test endpoint.")

			ampEndpointHandler(recorder, request, nil)
			assert.Equal(t, recorder.Code, http.StatusOK, "Endpoint should return 200 OK.")

			var actualAmpResp AmpResponse
			var expectedAmpResp AmpResponse
			assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &actualAmpResp), "Unable to unmarshal actual AmpResponse.")
			assert.NoError(t, json.Unmarshal(test.ExpectedAmpResponse, &expectedAmpResp), "Unable to unmarshal expected AmpResponse.")

			// validate modules data separately, because it has dynamic data
			if expectedAmpResp.ORTB2.Ext.Prebid == nil {
				assert.Nil(t, actualAmpResp.ORTB2.Ext.Prebid, "AmpResponse.ortb2.ext.prebid expected to be nil.")
			} else {
				hookexecution.AssertEqualModulesData(t, expectedAmpResp.ORTB2.Ext.Prebid.Modules, actualAmpResp.ORTB2.Ext.Prebid.Modules)
			}

			// reset modules to validate amp responses
			actualAmpResp.ORTB2.Ext.Prebid = nil
			expectedAmpResp.ORTB2.Ext.Prebid = nil
			assert.Equal(t, expectedAmpResp, actualAmpResp, "Invalid AMP Response.")

			// Close servers regardless if the test case was run or not
			for _, mockBidServer := range mockBidServers {
				mockBidServer.Close()
			}
			mockCurrencyRatesServer.Close()
		})
	}
}

func TestSendAmpResponse_LogsErrors(t *testing.T) {
	testCases := []struct {
		description    string
		expectedErrors []error
		expectedStatus int
		writer         http.ResponseWriter
		request        *openrtb2.BidRequest
		response       *openrtb2.BidResponse
		hookExecutor   hookexecution.HookStageExecutor
	}{
		{
			description: "Error logged when bid.ext unmarshal fails",
			expectedErrors: []error{
				errors.New("Critical error while unpacking AMP targets: unexpected end of JSON input"),
			},
			expectedStatus: http.StatusInternalServerError,
			writer:         httptest.NewRecorder(),
			request:        &openrtb2.BidRequest{ID: "some-id", Test: 1},
			response: &openrtb2.BidResponse{ID: "some-id", SeatBid: []openrtb2.SeatBid{
				{Bid: []openrtb2.Bid{{Ext: json.RawMessage(`"hb_cache_id`)}}},
			}},
			hookExecutor: &hookexecution.EmptyHookExecutor{},
		},
		{
			description: "Error logged when test mode activated but no debug present in response",
			expectedErrors: []error{
				errors.New("test set on request but debug not present in response"),
			},
			expectedStatus: 0,
			writer:         httptest.NewRecorder(),
			request:        &openrtb2.BidRequest{ID: "some-id", Test: 1},
			response:       &openrtb2.BidResponse{ID: "some-id", Ext: json.RawMessage("{}")},
			hookExecutor:   &hookexecution.EmptyHookExecutor{},
		},
		{
			description: "Error logged when response encoding fails",
			expectedErrors: []error{
				errors.New("/openrtb2/amp Failed to send response: failed writing response"),
			},
			expectedStatus: 0,
			writer:         errorResponseWriter{},
			request:        &openrtb2.BidRequest{ID: "some-id", Test: 1},
			response:       &openrtb2.BidResponse{ID: "some-id", Ext: json.RawMessage(`{"debug": {}}`)},
			hookExecutor:   &hookexecution.EmptyHookExecutor{},
		},
		{
			description: "Error logged if hook enrichment returns warnings",
			expectedErrors: []error{
				errors.New("Value is not a string: 1"),
				errors.New("Value is not a boolean: active"),
			},
			expectedStatus: 0,
			writer:         httptest.NewRecorder(),
			request:        &openrtb2.BidRequest{ID: "some-id", Ext: json.RawMessage(`{"prebid": {"debug": "active", "trace": 1}}`)},
			response:       &openrtb2.BidResponse{ID: "some-id", Ext: json.RawMessage("{}")},
			hookExecutor: &mockStageExecutor{
				outcomes: []hookexecution.StageOutcome{
					{
						Entity: "bid-request",
						Stage:  hooks.StageBidderRequest.String(),
						Groups: []hookexecution.GroupOutcome{
							{
								InvocationResults: []hookexecution.HookOutcome{
									{
										HookID: hookexecution.HookID{
											ModuleCode:   "foobar",
											HookImplCode: "foo",
										},
										Status:   hookexecution.StatusSuccess,
										Action:   hookexecution.ActionNone,
										Warnings: []string{"warning message"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			labels := metrics.Labels{}
			ao := analytics.AmpObject{}
			account := &config.Account{DebugAllow: true}
			reqWrapper := openrtb_ext.RequestWrapper{BidRequest: test.request}

			labels, ao = sendAmpResponse(test.writer, test.hookExecutor, test.response, &reqWrapper, account, labels, ao, nil)

			assert.Equal(t, ao.Errors, test.expectedErrors, "Invalid errors.")
			assert.Equal(t, test.expectedStatus, ao.Status, "Invalid HTTP response status.")
		})
	}
}

type errorResponseWriter struct{}

func (e errorResponseWriter) Header() http.Header {
	return http.Header{}
}

func (e errorResponseWriter) Write(bytes []byte) (int, error) {
	return 0, errors.New("failed writing response")
}

func (e errorResponseWriter) WriteHeader(statusCode int) {}
