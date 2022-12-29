package endpoints

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/usersync"
	"github.com/stretchr/testify/assert"

	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	metricsConf "github.com/prebid/prebid-server/metrics/config"
)

func TestSetUIDEndpoint(t *testing.T) {
	testCases := []struct {
		uri                    string
		syncersBidderNameToKey map[string]string
		existingSyncs          map[string]string
		gdprAllowsHostCookies  bool
		gdprReturnsError       bool
		gdprMalformed          bool
		expectedSyncs          map[string]string
		expectedBody           string
		expectedStatusCode     int
		expectedHeaders        map[string]string
		description            string
	}{
		{
			uri:                    "/setuid?bidder=pubmatic&uid=123",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          map[string]string{"pubmatic": "123"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "text/html", "Content-Length": "0"},
			description:            "Set uid for valid bidder",
		},
		{
			uri:                    "/setuid?bidder=adnxs&uid=123",
			syncersBidderNameToKey: map[string]string{"appnexus": "adnxs"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          map[string]string{"adnxs": "123"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "text/html", "Content-Length": "0"},
			description:            "Set uid for valid bidder with different key",
		},
		{
			uri:                    "/setuid?bidder=unsupported-bidder&uid=123",
			syncersBidderNameToKey: map[string]string{},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          nil,
			expectedStatusCode:     http.StatusBadRequest,
			expectedBody:           "The bidder name provided is not supported by Prebid Server",
			description:            "Don't set uid for an unsupported bidder",
		},
		{
			uri:                    "/setuid?bidder=&uid=123",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          nil,
			expectedStatusCode:     http.StatusBadRequest,
			expectedBody:           `"bidder" query param is required`,
			description:            "Don't set uid for an empty bidder",
		},
		{
			uri:                    "/setuid?bidder=unsupported-bidder&uid=123",
			syncersBidderNameToKey: map[string]string{},
			existingSyncs:          map[string]string{"pubmatic": "1234"},
			gdprAllowsHostCookies:  true,
			expectedSyncs:          nil,
			expectedStatusCode:     http.StatusBadRequest,
			expectedBody:           "The bidder name provided is not supported by Prebid Server",
			description: "No need to set existing syncs back in response for a request " +
				"to set uid for an unsupported bidder",
		},
		{
			uri:                    "/setuid?bidder=&uid=123",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          map[string]string{"pubmatic": "1234"},
			gdprAllowsHostCookies:  true,
			expectedSyncs:          nil,
			expectedStatusCode:     http.StatusBadRequest,
			expectedBody:           `"bidder" query param is required`,
			description: "No need to set existing syncs back in response for a request " +
				"to set uid for an empty bidder",
		},
		{
			uri:                    "/setuid?bidder=pubmatic",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          map[string]string{"pubmatic": "1234"},
			gdprAllowsHostCookies:  true,
			expectedSyncs:          map[string]string{},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "text/html", "Content-Length": "0"},
			description:            "Unset uid for a bidder if the request contains an empty uid for that bidder",
		},
		{
			uri:                    "/setuid?bidder=pubmatic&uid=123",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          map[string]string{"rubicon": "def"},
			gdprAllowsHostCookies:  true,
			expectedSyncs:          map[string]string{"pubmatic": "123", "rubicon": "def"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "text/html", "Content-Length": "0"},
			description:            "Add the uid for the requested bidder to the list of existing syncs",
		},
		{
			uri:                    "/setuid?bidder=pubmatic&uid=123&gdpr=0",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          map[string]string{"pubmatic": "123"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "text/html", "Content-Length": "0"},
			description:            "Don't care about GDPR consent if GDPR is set to 0",
		},
		{
			uri:                    "/setuid?uid=123",
			syncersBidderNameToKey: map[string]string{"appnexus": "appnexus"},
			existingSyncs:          nil,
			expectedSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedStatusCode:     http.StatusBadRequest,
			expectedBody:           `"bidder" query param is required`,
			description:            "Return an error if the bidder param is missing from the request",
		},
		{
			uri:                    "/setuid?bidder=appnexus&uid=123&gdpr=2",
			syncersBidderNameToKey: map[string]string{"appnexus": "appnexus"},
			existingSyncs:          nil,
			expectedSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedStatusCode:     http.StatusBadRequest,
			expectedBody:           "the gdpr query param must be either 0 or 1. You gave 2",
			description:            "Return an error if GDPR is set to anything else other that 0 or 1",
		},
		{
			uri:                    "/setuid?bidder=appnexus&uid=123&gdpr=1",
			syncersBidderNameToKey: map[string]string{"appnexus": "appnexus"},
			existingSyncs:          nil,
			expectedSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedStatusCode:     http.StatusBadRequest,
			expectedBody:           "gdpr_consent is required when gdpr=1",
			description:            "Return an error if GDPR is set to 1 but GDPR consent string is missing",
		},
		{
			uri: "/setuid?bidder=pubmatic&uid=123&gdpr_consent=" +
				"BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          nil,
			expectedSyncs:          nil,
			gdprReturnsError:       true,
			expectedStatusCode:     http.StatusBadRequest,
			expectedBody: "No global vendor list was available to interpret this consent string. " +
				"If this is a new, valid version, it should become available soon.",
			description: "Return an error if the GDPR string is either malformed or using a newer version that isn't yet supported",
		},
		{
			uri: "/setuid?bidder=pubmatic&uid=123&gdpr=1&gdpr_consent=" +
				"BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          nil,
			expectedSyncs:          nil,
			expectedStatusCode:     http.StatusUnavailableForLegalReasons,
			expectedBody:           "The gdpr_consent string prevents cookies from being saved",
			description:            "Shouldn't set uid for a bidder if it is not allowed by the GDPR consent string",
		},
		{
			uri: "/setuid?bidder=pubmatic&uid=123&gdpr=1&gdpr_consent=" +
				"BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			gdprAllowsHostCookies:  true,
			existingSyncs:          nil,
			expectedSyncs:          map[string]string{"pubmatic": "123"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "text/html", "Content-Length": "0"},
			description:            "Should set uid for a bidder that is allowed by the GDPR consent string",
		},
		{
			uri: "/setuid?bidder=pubmatic&uid=123&gdpr=1&gdpr_consent=" +
				"malformed",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			gdprAllowsHostCookies:  true,
			gdprMalformed:          true,
			existingSyncs:          nil,
			expectedStatusCode:     http.StatusBadRequest,
			expectedBody:           "gdpr_consent was invalid. malformed consent string malformed: some error",
			description:            "Should return an error if GDPR consent string is malformed",
		},
		{
			uri:                    "/setuid?bidder=pubmatic&uid=123&f=b",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          map[string]string{"pubmatic": "123"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "text/html", "Content-Length": "0"},
			description:            "Set uid for valid bidder with iframe format",
		},
		{
			uri:                    "/setuid?bidder=pubmatic&uid=123&f=i",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          map[string]string{"pubmatic": "123"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "image/png", "Content-Length": "86"},
			description:            "Set uid for valid bidder with redirect format",
		},
		{
			uri:                    "/setuid?bidder=pubmatic&uid=123&f=x",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          nil,
			expectedStatusCode:     http.StatusBadRequest,
			expectedBody:           `"f" query param is invalid. must be "b" or "i"`,
			description:            "Set uid for valid bidder with invalid format",
		},
		{
			uri:                    "/setuid?bidder=pubmatic&uid=123&account=valid_acct",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          map[string]string{"pubmatic": "123"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "text/html", "Content-Length": "0"},
			description:            "Set uid for valid bidder with valid account provided",
		},
		{
			uri:                    "/setuid?bidder=pubmatic&uid=123&account=disabled_acct",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          nil,
			expectedStatusCode:     http.StatusBadRequest,
			expectedBody:           "account is disabled, please reach out to the prebid server host",
			description:            "Set uid for valid bidder with valid disabled account provided",
		},
	}

	analytics := analyticsConf.NewPBSAnalytics(&config.Analytics{})
	metrics := &metricsConf.NilMetricsEngine{}

	for _, test := range testCases {
		response := doRequest(makeRequest(test.uri, test.existingSyncs), analytics, metrics,
			test.syncersBidderNameToKey, test.gdprAllowsHostCookies, test.gdprReturnsError, test.gdprMalformed, false)
		assert.Equal(t, test.expectedStatusCode, response.Code, "Test Case: %s. /setuid returned unexpected error code", test.description)

		if test.expectedSyncs != nil {
			assertHasSyncs(t, test.description, response, test.expectedSyncs)
		} else {
			assert.Equal(t, "", response.Header().Get("Set-Cookie"), "Test Case: %s. /setuid returned unexpected cookie", test.description)
		}

		if test.expectedBody != "" {
			assert.Equal(t, test.expectedBody, response.Body.String(), "Test Case: %s. /setuid returned unexpected message", test.description)
		}

		// compare header values, except for the cookies
		responseHeaders := map[string]string{}
		for k, v := range response.Result().Header {
			if k != "Set-Cookie" {
				responseHeaders[k] = v[0]
			}
		}
		if test.expectedHeaders == nil {
			test.expectedHeaders = map[string]string{}
		}
		assert.Equal(t, test.expectedHeaders, responseHeaders, test.description+":headers")
	}
}

func TestSetUIDEndpointMetrics(t *testing.T) {
	cookieWithOptOut := usersync.NewCookie()
	cookieWithOptOut.SetOptOut(true)

	testCases := []struct {
		description            string
		uri                    string
		cookies                []*usersync.Cookie
		syncersBidderNameToKey map[string]string
		gdprAllowsHostCookies  bool
		cfgAccountRequired     bool
		expectedResponseCode   int
		expectedMetrics        func(*metrics.MetricsEngineMock)
		expectedAnalytics      func(*MockAnalytics)
	}{
		{
			description:            "Success - Sync",
			uri:                    "/setuid?bidder=pubmatic&uid=123",
			cookies:                []*usersync.Cookie{},
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			gdprAllowsHostCookies:  true,
			expectedResponseCode:   200,
			expectedMetrics: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSetUid", metrics.SetUidOK).Once()
				m.On("RecordSyncerSet", "pubmatic", metrics.SyncerSetUidOK).Once()
			},
			expectedAnalytics: func(a *MockAnalytics) {
				expected := analytics.SetUIDObject{
					Status:  200,
					Bidder:  "pubmatic",
					UID:     "123",
					Errors:  []error{},
					Success: true,
				}
				a.On("LogSetUIDObject", &expected).Once()
			},
		},
		{
			description:            "Success - Unsync",
			uri:                    "/setuid?bidder=pubmatic&uid=",
			cookies:                []*usersync.Cookie{},
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			gdprAllowsHostCookies:  true,
			expectedResponseCode:   200,
			expectedMetrics: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSetUid", metrics.SetUidOK).Once()
				m.On("RecordSyncerSet", "pubmatic", metrics.SyncerSetUidCleared).Once()
			},
			expectedAnalytics: func(a *MockAnalytics) {
				expected := analytics.SetUIDObject{
					Status:  200,
					Bidder:  "pubmatic",
					UID:     "",
					Errors:  []error{},
					Success: true,
				}
				a.On("LogSetUIDObject", &expected).Once()
			},
		},
		{
			description:            "Cookie Opted Out",
			uri:                    "/setuid?bidder=pubmatic&uid=123",
			cookies:                []*usersync.Cookie{cookieWithOptOut},
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			gdprAllowsHostCookies:  true,
			expectedResponseCode:   401,
			expectedMetrics: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSetUid", metrics.SetUidOptOut).Once()
			},
			expectedAnalytics: func(a *MockAnalytics) {
				expected := analytics.SetUIDObject{
					Status:  401,
					Bidder:  "",
					UID:     "",
					Errors:  []error{},
					Success: false,
				}
				a.On("LogSetUIDObject", &expected).Once()
			},
		},
		{
			description:            "Unknown Syncer Key",
			uri:                    "/setuid?bidder=pubmatic&uid=123",
			cookies:                []*usersync.Cookie{},
			syncersBidderNameToKey: map[string]string{},
			gdprAllowsHostCookies:  true,
			expectedResponseCode:   400,
			expectedMetrics: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSetUid", metrics.SetUidSyncerUnknown).Once()
			},
			expectedAnalytics: func(a *MockAnalytics) {
				expected := analytics.SetUIDObject{
					Status:  400,
					Bidder:  "",
					UID:     "",
					Errors:  []error{errors.New("The bidder name provided is not supported by Prebid Server")},
					Success: false,
				}
				a.On("LogSetUIDObject", &expected).Once()
			},
		},
		{
			description:            "Unknown Format",
			uri:                    "/setuid?bidder=pubmatic&uid=123&f=z",
			cookies:                []*usersync.Cookie{},
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			gdprAllowsHostCookies:  true,
			expectedResponseCode:   400,
			expectedMetrics: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSetUid", metrics.SetUidBadRequest).Once()
			},
			expectedAnalytics: func(a *MockAnalytics) {
				expected := analytics.SetUIDObject{
					Status:  400,
					Bidder:  "pubmatic",
					UID:     "",
					Errors:  []error{errors.New(`"f" query param is invalid. must be "b" or "i"`)},
					Success: false,
				}
				a.On("LogSetUIDObject", &expected).Once()
			},
		},
		{
			description:            "Prevented By GDPR - Invalid Consent String",
			uri:                    "/setuid?bidder=pubmatic&uid=123&gdpr=1",
			cookies:                []*usersync.Cookie{},
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			gdprAllowsHostCookies:  true,
			expectedResponseCode:   400,
			expectedMetrics: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSetUid", metrics.SetUidBadRequest).Once()
			},
			expectedAnalytics: func(a *MockAnalytics) {
				expected := analytics.SetUIDObject{
					Status:  400,
					Bidder:  "pubmatic",
					UID:     "",
					Errors:  []error{errors.New("gdpr_consent is required when gdpr=1")},
					Success: false,
				}
				a.On("LogSetUIDObject", &expected).Once()
			},
		},
		{
			description:            "Prevented By GDPR - Permission Denied By Consent String",
			uri:                    "/setuid?bidder=pubmatic&uid=123&gdpr=1&gdpr_consent=any",
			cookies:                []*usersync.Cookie{},
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			gdprAllowsHostCookies:  false,
			expectedResponseCode:   451,
			expectedMetrics: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSetUid", metrics.SetUidGDPRHostCookieBlocked).Once()
			},
			expectedAnalytics: func(a *MockAnalytics) {
				expected := analytics.SetUIDObject{
					Status:  451,
					Bidder:  "pubmatic",
					UID:     "",
					Errors:  []error{errors.New("The gdpr_consent string prevents cookies from being saved")},
					Success: false,
				}
				a.On("LogSetUIDObject", &expected).Once()
			},
		},
		{
			description:            "Blocked account",
			uri:                    "/setuid?bidder=pubmatic&uid=123&account=blocked_acct",
			cookies:                []*usersync.Cookie{},
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			gdprAllowsHostCookies:  true,
			expectedResponseCode:   400,
			expectedMetrics: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSetUid", metrics.SetUidAccountBlocked).Once()
			},
			expectedAnalytics: func(a *MockAnalytics) {
				expected := analytics.SetUIDObject{
					Status:  400,
					Bidder:  "pubmatic",
					UID:     "",
					Errors:  []error{errCookieSyncAccountBlocked},
					Success: false,
				}
				a.On("LogSetUIDObject", &expected).Once()
			},
		},
		{
			description:            "Invalid account",
			uri:                    "/setuid?bidder=pubmatic&uid=123&account=unknown",
			cookies:                []*usersync.Cookie{},
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			gdprAllowsHostCookies:  true,
			cfgAccountRequired:     true,
			expectedResponseCode:   400,
			expectedMetrics: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSetUid", metrics.SetUidAccountInvalid).Once()
			},
			expectedAnalytics: func(a *MockAnalytics) {
				expected := analytics.SetUIDObject{
					Status:  400,
					Bidder:  "pubmatic",
					UID:     "",
					Errors:  []error{errCookieSyncAccountInvalid},
					Success: false,
				}
				a.On("LogSetUIDObject", &expected).Once()
			},
		},
		{
			description:            "Malformed account",
			uri:                    "/setuid?bidder=pubmatic&uid=123&account=malformed_acct",
			cookies:                []*usersync.Cookie{},
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			gdprAllowsHostCookies:  true,
			cfgAccountRequired:     true,
			expectedResponseCode:   400,
			expectedMetrics: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSetUid", metrics.SetUidAccountConfigMalformed).Once()
			},
			expectedAnalytics: func(a *MockAnalytics) {
				expected := analytics.SetUIDObject{
					Status:  400,
					Bidder:  "pubmatic",
					UID:     "",
					Errors:  []error{errCookieSyncAccountConfigMalformed},
					Success: false,
				}
				a.On("LogSetUIDObject", &expected).Once()
			},
		},
		{
			description:            "Invalid JSON account",
			uri:                    "/setuid?bidder=pubmatic&uid=123&account=invalid_json_acct",
			cookies:                []*usersync.Cookie{},
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			gdprAllowsHostCookies:  true,
			cfgAccountRequired:     true,
			expectedResponseCode:   400,
			expectedMetrics: func(m *metrics.MetricsEngineMock) {
				m.On("RecordSetUid", metrics.SetUidBadRequest).Once()
			},
			expectedAnalytics: func(a *MockAnalytics) {
				expected := analytics.SetUIDObject{
					Status:  400,
					Bidder:  "pubmatic",
					UID:     "",
					Errors:  []error{errors.New("Invalid JSON Patch")},
					Success: false,
				}
				a.On("LogSetUIDObject", &expected).Once()
			},
		},
	}

	for _, test := range testCases {
		analyticsEngine := &MockAnalytics{}
		test.expectedAnalytics(analyticsEngine)

		metricsEngine := &metrics.MetricsEngineMock{}
		test.expectedMetrics(metricsEngine)

		req := httptest.NewRequest("GET", test.uri, nil)
		for _, v := range test.cookies {
			addCookie(req, v)
		}
		response := doRequest(req, analyticsEngine, metricsEngine, test.syncersBidderNameToKey, test.gdprAllowsHostCookies, false, false, test.cfgAccountRequired)

		assert.Equal(t, test.expectedResponseCode, response.Code, test.description)
		analyticsEngine.AssertExpectations(t)
		metricsEngine.AssertExpectations(t)
	}
}

func TestOptedOut(t *testing.T) {
	request := httptest.NewRequest("GET", "/setuid?bidder=pubmatic&uid=123", nil)
	cookie := usersync.NewCookie()
	cookie.SetOptOut(true)
	addCookie(request, cookie)
	syncersBidderNameToKey := map[string]string{"pubmatic": "pubmatic"}
	analytics := analyticsConf.NewPBSAnalytics(&config.Analytics{})
	metrics := &metricsConf.NilMetricsEngine{}
	response := doRequest(request, analytics, metrics, syncersBidderNameToKey, true, false, false, false)

	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestSiteCookieCheck(t *testing.T) {
	testCases := []struct {
		ua             string
		expectedResult bool
		description    string
	}{
		{
			ua:             "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.142 Safari/537.36",
			expectedResult: true,
			description:    "Should return true for a valid chrome version",
		},
		{
			ua:             "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3770.142 Safari/537.36",
			expectedResult: false,
			description:    "Should return false for chrome version below than the supported min version",
		},
	}

	for _, test := range testCases {
		assert.Equal(t, test.expectedResult, siteCookieCheck(test.ua), test.description)
	}
}

func TestGetResponseFormat(t *testing.T) {
	testCases := []struct {
		urlValues      url.Values
		syncer         usersync.Syncer
		expectedFormat string
		expectedError  string
		description    string
	}{
		{
			urlValues:      url.Values{},
			syncer:         fakeSyncer{key: "a", defaultSyncType: usersync.SyncTypeIFrame},
			expectedFormat: "b",
			description:    "parameter not provided, use default sync type iframe",
		},
		{
			urlValues:      url.Values{},
			syncer:         fakeSyncer{key: "a", defaultSyncType: usersync.SyncTypeRedirect},
			expectedFormat: "i",
			description:    "parameter not provided, use default sync type redirect",
		},
		{
			urlValues:      url.Values{},
			syncer:         fakeSyncer{key: "a", defaultSyncType: usersync.SyncType("invalid")},
			expectedFormat: "",
			description:    "parameter not provided,  default sync type is invalid",
		},
		{
			urlValues:      url.Values{"f": []string{"b"}},
			syncer:         fakeSyncer{key: "a", defaultSyncType: usersync.SyncTypeRedirect},
			expectedFormat: "b",
			description:    "parameter given as `b`, default sync type is opposite",
		},
		{
			urlValues:      url.Values{"f": []string{"B"}},
			syncer:         fakeSyncer{key: "a", defaultSyncType: usersync.SyncTypeRedirect},
			expectedFormat: "b",
			description:    "parameter given as `b`, default sync type is opposite - case insensitive",
		},
		{
			urlValues:      url.Values{"f": []string{"i"}},
			syncer:         fakeSyncer{key: "a", defaultSyncType: usersync.SyncTypeIFrame},
			expectedFormat: "i",
			description:    "parameter given as `b`, default sync type is opposite",
		},
		{
			urlValues:      url.Values{"f": []string{"I"}},
			syncer:         fakeSyncer{key: "a", defaultSyncType: usersync.SyncTypeIFrame},
			expectedFormat: "i",
			description:    "parameter given as `b`, default sync type is opposite - case insensitive",
		},
		{
			urlValues:     url.Values{"f": []string{"x"}},
			syncer:        fakeSyncer{key: "a", defaultSyncType: usersync.SyncTypeIFrame},
			expectedError: `"f" query param is invalid. must be "b" or "i"`,
			description:   "parameter given invalid",
		},
		{
			urlValues:      url.Values{"f": []string{}},
			syncer:         fakeSyncer{key: "a", defaultSyncType: usersync.SyncTypeRedirect},
			expectedFormat: "i",
			description:    "parameter given is empty (by slice), use default sync type redirect",
		},
		{
			urlValues:      url.Values{"f": []string{""}},
			syncer:         fakeSyncer{key: "a", defaultSyncType: usersync.SyncTypeRedirect},
			expectedFormat: "i",
			description:    "parameter given is empty (by empty item), use default sync type redirect",
		},
	}

	for _, test := range testCases {
		result, err := getResponseFormat(test.urlValues, test.syncer)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
			assert.Equal(t, test.expectedFormat, result, test.description+":result")
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
			assert.Empty(t, result, test.description+":result")
		}
	}
}

func assertHasSyncs(t *testing.T, testCase string, resp *httptest.ResponseRecorder, syncs map[string]string) {
	t.Helper()
	cookie := parseCookieString(t, resp)

	assert.Equal(t, len(syncs), len(cookie.GetUIDs()), "Test Case: %s. /setuid response doesn't contain expected number of syncs", testCase)

	for bidder, uid := range syncs {
		assert.True(t, cookie.HasLiveSync(bidder), "Test Case: %s. /setuid response cookie doesn't contain uid for bidder: %s", testCase, bidder)
		actualUID, _, _ := cookie.GetUID(bidder)
		assert.Equal(t, uid, actualUID, "Test Case: %s. /setuid response cookie doesn't contain correct uid for bidder: %s", testCase, bidder)
	}
}

func makeRequest(uri string, existingSyncs map[string]string) *http.Request {
	request := httptest.NewRequest("GET", uri, nil)
	if len(existingSyncs) > 0 {
		pbsCookie := usersync.NewCookie()
		for key, value := range existingSyncs {
			pbsCookie.TrySync(key, value)
		}
		addCookie(request, pbsCookie)
	}
	return request
}

func doRequest(req *http.Request, analytics analytics.PBSAnalyticsModule, metrics metrics.MetricsEngine, syncersBidderNameToKey map[string]string, gdprAllowsHostCookies, gdprReturnsError, gdprReturnsMalformedError, cfgAccountRequired bool) *httptest.ResponseRecorder {
	cfg := config.Configuration{
		AccountRequired: cfgAccountRequired,
		BlacklistedAcctMap: map[string]bool{
			"blocked_acct": true,
		},
	}
	cfg.MarshalAccountDefaults()

	query := req.URL.Query()

	perms := &fakePermsSetUID{
		allowHost:           gdprAllowsHostCookies,
		consent:             query.Get("gdpr_consent"),
		errorHost:           gdprReturnsError,
		errorMalformed:      gdprReturnsMalformedError,
		personalInfoAllowed: true,
	}
	gdprPermsBuilder := fakePermissionsBuilder{
		permissions: perms,
	}.Builder
	tcf2ConfigBuilder := fakeTCF2ConfigBuilder{
		cfg: gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
	}.Builder

	syncersByBidder := make(map[string]usersync.Syncer)
	for bidderName, syncerKey := range syncersBidderNameToKey {
		syncersByBidder[bidderName] = fakeSyncer{key: syncerKey, defaultSyncType: usersync.SyncTypeIFrame}
	}

	fakeAccountsFetcher := FakeAccountsFetcher{AccountData: map[string]json.RawMessage{
		"valid_acct":        json.RawMessage(`{"disabled":false}`),
		"disabled_acct":     json.RawMessage(`{"disabled":true}`),
		"malformed_acct":    json.RawMessage(`{"disabled":"malformed"}`),
		"invalid_json_acct": json.RawMessage(`{"}`),
	}}

	endpoint := NewSetUIDEndpoint(&cfg, syncersByBidder, gdprPermsBuilder, tcf2ConfigBuilder, analytics, fakeAccountsFetcher, metrics)
	response := httptest.NewRecorder()
	endpoint(response, req, nil)
	return response
}

func addCookie(req *http.Request, cookie *usersync.Cookie) {
	req.AddCookie(cookie.ToHTTPCookie(time.Duration(1) * time.Hour))
}

func parseCookieString(t *testing.T, response *httptest.ResponseRecorder) *usersync.Cookie {
	cookieString := response.Header().Get("Set-Cookie")
	parser := regexp.MustCompile("uids=(.*?);")
	res := parser.FindStringSubmatch(cookieString)
	assert.Equal(t, 2, len(res))
	httpCookie := http.Cookie{
		Name:  "uids",
		Value: res[1],
	}
	return usersync.ParseCookie(&httpCookie)
}

type fakePermissionsBuilder struct {
	permissions gdpr.Permissions
}

func (fpb fakePermissionsBuilder) Builder(gdpr.TCF2ConfigReader, gdpr.RequestInfo) gdpr.Permissions {
	return fpb.permissions
}

type fakeTCF2ConfigBuilder struct {
	cfg gdpr.TCF2ConfigReader
}

func (fcr fakeTCF2ConfigBuilder) Builder(hostConfig config.TCF2, accountConfig config.AccountGDPR) gdpr.TCF2ConfigReader {
	return fcr.cfg
}

type fakePermsSetUID struct {
	allowHost           bool
	consent             string
	errorHost           bool
	errorMalformed      bool
	personalInfoAllowed bool
}

func (g *fakePermsSetUID) HostCookiesAllowed(ctx context.Context) (bool, error) {
	if g.errorMalformed {
		return g.allowHost, &gdpr.ErrorMalformedConsent{Consent: g.consent, Cause: errors.New("some error")}
	}
	if g.errorHost {
		return g.allowHost, errors.New("something went wrong")
	}
	return g.allowHost, nil
}

func (g *fakePermsSetUID) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName) (bool, error) {
	return false, nil
}

func (g *fakePermsSetUID) AuctionActivitiesAllowed(ctx context.Context, bidderCoreName openrtb_ext.BidderName, bidder openrtb_ext.BidderName) (permissions gdpr.AuctionPermissions, err error) {
	return gdpr.AuctionPermissions{
		AllowBidRequest: g.personalInfoAllowed,
		PassGeo:         g.personalInfoAllowed,
		PassID:          g.personalInfoAllowed,
	}, nil
}

type fakeSyncer struct {
	key             string
	defaultSyncType usersync.SyncType
}

func (s fakeSyncer) Key() string {
	return s.key
}

func (s fakeSyncer) DefaultSyncType() usersync.SyncType {
	return s.defaultSyncType
}

func (s fakeSyncer) SupportsType(syncTypes []usersync.SyncType) bool {
	return true
}

func (s fakeSyncer) GetSync(syncTypes []usersync.SyncType, privacyPolicies privacy.Policies) (usersync.Sync, error) {
	return usersync.Sync{}, nil
}
