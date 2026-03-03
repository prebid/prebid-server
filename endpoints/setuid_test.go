package endpoints

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/analytics"
	analyticsBuild "github.com/prebid/prebid-server/v3/analytics/build"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/usersync"
	"github.com/stretchr/testify/assert"

	metricsConf "github.com/prebid/prebid-server/v3/metrics/config"
)

func TestSetUIDEndpoint(t *testing.T) {
	testCases := []struct {
		uri                    string
		syncersBidderNameToKey map[string]string
		existingSyncs          map[string]string
		gdprAllowsHostCookies  bool
		gdprReturnsError       bool
		gdprMalformed          bool
		formatOverride         string
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
			uri:                    "/setuid?bidder=PUBMATIC&uid=123",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          map[string]string{"pubmatic": "123"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "text/html", "Content-Length": "0"},
			description:            "Set uid for valid bidder case insensitive",
		},
		{
			uri:                    "/setuid?bidder=appnexus&uid=123",
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
			expectedBody:           "GDPR consent is required when gdpr signal equals 1",
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
			uri:                    "/setuid?bidder=pubmatic&uid=123&gpp_sid=2,4&gpp=DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			gdprAllowsHostCookies:  true,
			existingSyncs:          nil,
			expectedSyncs:          map[string]string{"pubmatic": "123"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "text/html", "Content-Length": "0"},
			description:            "Sets uid for a bidder allowed by GDPR consent string in the GPP query field",
		},
		{
			uri: "/setuid?bidder=pubmatic&uid=123&gpp_sid=2,4&gpp=DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA" +
				"&gdpr=1&gdpr_consent=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			gdprAllowsHostCookies:  true,
			existingSyncs:          nil,
			expectedSyncs:          map[string]string{"pubmatic": "123"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "text/html", "Content-Length": "0"},
			description:            "GPP value will be used over the one found in the deprecated GDPR consent field for iframe format",
		},
		{
			uri: "/setuid?f=i&bidder=pubmatic&uid=123&gpp_sid=2,4&gpp=DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA" +
				"&gdpr=1&gdpr_consent=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			gdprAllowsHostCookies:  true,
			existingSyncs:          nil,
			expectedSyncs:          map[string]string{"pubmatic": "123"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "image/png", "Content-Length": "86"},
			description:            "GPP value will be used over the one found in the deprecated GDPR consent field for redirect format",
		},
		{
			uri:                    "/setuid?bidder=pubmatic&uid=123&gdpr=1&gdpr_consent=malformed",
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
		{
			uri:                    "/setuid?bidder=pubmatic&uid=123&account=valid_acct_with_valid_activities_usersync_enabled",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          map[string]string{"pubmatic": "123"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "text/html", "Content-Length": "0"},
			description:            "Set uid for valid bidder with valid account provided with user sync allowed activity",
		},
		{
			uri:                    "/setuid?bidder=pubmatic&uid=123&account=valid_acct_with_valid_activities_usersync_disabled",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          nil,
			expectedStatusCode:     http.StatusUnavailableForLegalReasons,
			description:            "Set uid for valid bidder with valid account provided with user sync disallowed activity",
		},
		{
			uri:                    "/setuid?bidder=pubmatic&uid=123&account=valid_acct_with_invalid_activities",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          map[string]string{"pubmatic": "123"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "text/html", "Content-Length": "0"},
			description:            "Set uid for valid bidder with valid account provided with invalid user sync activity",
		},
		{
			description:            "gppsid-valid",
			uri:                    "/setuid?bidder=appnexus&uid=123&gpp_sid=100,101", // fake sids to avoid GDPR logic in this test
			syncersBidderNameToKey: map[string]string{"appnexus": "appnexus"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          map[string]string{"appnexus": "123"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Type": "text/html", "Content-Length": "0"},
		},
		{
			description:            "gppsid-malformed",
			uri:                    "/setuid?bidder=appnexus&uid=123&gpp_sid=malformed",
			syncersBidderNameToKey: map[string]string{"appnexus": "appnexus"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			expectedSyncs:          nil,
			expectedStatusCode:     http.StatusBadRequest,
			expectedBody:           "invalid gpp_sid encoding, must be a csv list of integers",
		},
		{
			uri:                    "/setuid?bidder=pubmatic&uid=123",
			syncersBidderNameToKey: map[string]string{"pubmatic": "pubmatic"},
			existingSyncs:          nil,
			gdprAllowsHostCookies:  true,
			formatOverride:         "i",
			expectedSyncs:          map[string]string{"pubmatic": "123"},
			expectedStatusCode:     http.StatusOK,
			expectedHeaders:        map[string]string{"Content-Length": "86", "Content-Type": "image/png"},
			description:            "Format not provided in URL, but formatOverride is defined",
		},
	}

	analytics := analyticsBuild.New(&config.Analytics{})
	metrics := &metricsConf.NilMetricsEngine{}

	for _, test := range testCases {
		response := doRequest(makeRequest(test.uri, test.existingSyncs), analytics, metrics,
			test.syncersBidderNameToKey, test.gdprAllowsHostCookies, test.gdprReturnsError, test.gdprMalformed, false, 0, nil, test.formatOverride)
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

func TestSetUIDPriorityEjection(t *testing.T) {
	decoder := usersync.Base64Decoder{}
	analytics := analyticsBuild.New(&config.Analytics{})
	syncersByBidder := map[string]string{
		"pubmatic":             "pubmatic",
		"syncer1":              "syncer1",
		"syncer2":              "syncer2",
		"syncer3":              "syncer3",
		"syncer4":              "syncer4",
		"mismatchedBidderName": "syncer5",
		"syncerToEject":        "syncerToEject",
	}

	testCases := []struct {
		description           string
		uri                   string
		givenExistingSyncs    []string
		givenPriorityGroups   [][]string
		givenMaxCookieSize    int
		expectedStatusCode    int
		expectedSyncer        string
		expectedUID           string
		expectedNumOfElements int
		expectedWarning       string
	}{
		{
			description:           "Cookie empty, expect bidder to be synced, no ejection",
			uri:                   "/setuid?bidder=pubmatic&uid=123",
			givenPriorityGroups:   [][]string{},
			givenMaxCookieSize:    500,
			expectedSyncer:        "pubmatic",
			expectedUID:           "123",
			expectedNumOfElements: 1,
			expectedStatusCode:    http.StatusOK,
		},
		{
			description:           "Cookie full, no priority groups, one ejection",
			uri:                   "/setuid?bidder=pubmatic&uid=123",
			givenExistingSyncs:    []string{"syncer1", "syncer2", "syncer3", "syncer4"},
			givenPriorityGroups:   [][]string{},
			givenMaxCookieSize:    500,
			expectedUID:           "123",
			expectedSyncer:        "pubmatic",
			expectedNumOfElements: 4,
			expectedStatusCode:    http.StatusOK,
		},
		{
			description:           "Cookie full, eject lowest priority element",
			uri:                   "/setuid?bidder=pubmatic&uid=123",
			givenExistingSyncs:    []string{"syncer2", "syncer3", "syncer4", "syncerToEject"},
			givenPriorityGroups:   [][]string{{"pubmatic", "syncer2", "syncer3", "syncer4"}, {"syncerToEject"}},
			givenMaxCookieSize:    500,
			expectedUID:           "123",
			expectedSyncer:        "pubmatic",
			expectedNumOfElements: 4,
			expectedStatusCode:    http.StatusOK,
		},
		{
			description:           "Cookie full, all elements same priority, one ejection",
			uri:                   "/setuid?bidder=pubmatic&uid=123",
			givenExistingSyncs:    []string{"syncer1", "syncer2", "syncer3", "syncer5"},
			givenPriorityGroups:   [][]string{{"pubmatic", "syncer1", "syncer2", "syncer3", "mismatchedBidderName"}},
			givenMaxCookieSize:    500,
			expectedUID:           "123",
			expectedSyncer:        "pubmatic",
			expectedNumOfElements: 4,
			expectedStatusCode:    http.StatusOK,
		},
		{
			description:         "There are only priority elements left, but the bidder being synced isn't one",
			uri:                 "/setuid?bidder=pubmatic&uid=123",
			givenExistingSyncs:  []string{"syncer1", "syncer2", "syncer3", "syncer4"},
			givenPriorityGroups: [][]string{{"syncer1", "syncer2", "syncer3", "syncer4"}},
			givenMaxCookieSize:  500,
			expectedStatusCode:  http.StatusOK,
			expectedWarning:     "Warning: syncer key is not a priority, and there are only priority elements left, cookie not updated",
		},
		{
			description:        "Uid that's trying to be synced is bigger than MaxCookieSize",
			uri:                "/setuid?bidder=pubmatic&uid=123",
			givenMaxCookieSize: 1,
			expectedStatusCode: http.StatusBadRequest,
		},
	}
	for _, test := range testCases {
		request := httptest.NewRequest("GET", test.uri, nil)

		// Cookie Set Up
		cookie := usersync.NewCookie()
		for _, key := range test.givenExistingSyncs {
			cookie.Sync(key, "111")
		}
		httpCookie, err := ToHTTPCookie(cookie)
		assert.NoError(t, err)
		request.AddCookie(httpCookie)

		// Make Request to /setuid
		response := doRequest(request, analytics, &metricsConf.NilMetricsEngine{}, syncersByBidder, true, false, false, false, test.givenMaxCookieSize, test.givenPriorityGroups, "")

		if test.expectedWarning != "" {
			assert.Equal(t, test.expectedWarning, response.Body.String(), test.description)
		} else if test.expectedSyncer != "" {
			// Get Cookie From Header
			var cookieHeader string
			for k, v := range response.Result().Header {
				if k == "Set-Cookie" {
					cookieHeader = v[0]
				}
			}
			encodedCookieValue := getUIDFromHeader(cookieHeader)

			// Check That Bidder On Request was Synced, it's UID matches, and that the right number of elements are present after ejection
			decodedCookie := decoder.Decode(encodedCookieValue)
			decodedCookieUIDs := decodedCookie.GetUIDs()

			assert.Equal(t, test.expectedUID, decodedCookieUIDs[test.expectedSyncer], test.description)
			assert.Equal(t, test.expectedNumOfElements, len(decodedCookieUIDs), test.description)

			// Specific test case handling where we eject the lowest priority element
			if len(test.givenPriorityGroups) == 2 {
				syncer := test.givenPriorityGroups[len(test.givenPriorityGroups)-1][0]
				_, syncerExists := decodedCookieUIDs[syncer]
				assert.False(t, syncerExists, test.description)
			}
		}
		assert.Equal(t, test.expectedStatusCode, response.Result().StatusCode, test.description)
	}
}

func TestParseSignalFromGPPSID(t *testing.T) {
	type testOutput struct {
		signal gdpr.Signal
		err    error
	}
	testCases := []struct {
		desc     string
		strSID   string
		expected testOutput
	}{
		{
			desc:   "Empty gpp_sid, expect gdpr.SignalAmbiguous",
			strSID: "",
			expected: testOutput{
				signal: gdpr.SignalAmbiguous,
				err:    nil,
			},
		},
		{
			desc:   "Malformed gpp_sid, expect gdpr.SignalAmbiguous",
			strSID: "malformed",
			expected: testOutput{
				signal: gdpr.SignalAmbiguous,
				err:    errors.New(`Error parsing gpp_sid strconv.ParseInt: parsing "malformed": invalid syntax`),
			},
		},
		{
			desc:   "Valid gpp_sid doesn't come with TCF2, expect gdpr.SignalNo",
			strSID: "6",
			expected: testOutput{
				signal: gdpr.SignalNo,
				err:    nil,
			},
		},
		{
			desc:   "Valid gpp_sid comes with TCF2, expect gdpr.SignalYes",
			strSID: "2",
			expected: testOutput{
				signal: gdpr.SignalYes,
				err:    nil,
			},
		},
	}
	for _, tc := range testCases {
		outSignal, outErr := parseSignalFromGppSidStr(tc.strSID)

		assert.Equal(t, tc.expected.signal, outSignal, tc.desc)
		assert.Equal(t, tc.expected.err, outErr, tc.desc)
	}
}

func TestParseConsentFromGppStr(t *testing.T) {
	type testOutput struct {
		gdprConsent string
		err         []error
	}
	testCases := []struct {
		desc       string
		inGppQuery string
		expected   testOutput
	}{
		{
			desc:       "Empty gpp field, expect empty GDPR consent",
			inGppQuery: "",
			expected: testOutput{
				gdprConsent: "",
				err:         nil,
			},
		},
		{
			desc:       "Malformed gpp field value, expect empty GDPR consent and error",
			inGppQuery: "malformed",
			expected: testOutput{
				gdprConsent: "",
				err:         []error{errors.New(`error parsing GPP header, header must have type=3`)},
			},
		},
		{
			desc:       "Valid gpp string comes with TCF2 in its gppConstants.SectionID's, expect non-empty GDPR consent",
			inGppQuery: "DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
			expected: testOutput{
				gdprConsent: "CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
				err:         nil,
			},
		},
		{
			desc:       "Valid gpp string doesn't come with TCF2 in its gppConstants.SectionID's, expect blank GDPR consent",
			inGppQuery: "DBABjw~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~1YNN",
			expected: testOutput{
				gdprConsent: "",
				err:         nil,
			},
		},
	}
	for _, tc := range testCases {
		outConsent, outErr := parseConsentFromGppStr(tc.inGppQuery)

		assert.Equal(t, tc.expected.gdprConsent, outConsent, tc.desc)
		assert.ElementsMatch(t, tc.expected.err, outErr, tc.desc)
	}
}

func TestParseGDPRFromGPP(t *testing.T) {
	type testOutput struct {
		reqInfo gdpr.RequestInfo
		err     error
	}
	type aTest struct {
		desc     string
		inUri    string
		expected testOutput
	}
	testGroups := []struct {
		groupDesc string
		testCases []aTest
	}{
		{
			groupDesc: "No gpp_sid nor gpp",
			testCases: []aTest{
				{
					desc:  "Input URL is mising gpp_sid and gpp, expect signal ambiguous and no error",
					inUri: "/setuid?bidder=pubmatic&uid=123",
					expected: testOutput{
						reqInfo: gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous},
						err:     nil,
					},
				},
			},
		},
		{
			groupDesc: "gpp only",
			testCases: []aTest{
				{
					desc:  "gpp is malformed, expect error",
					inUri: "/setuid?gpp=malformed",
					expected: testOutput{
						reqInfo: gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous},
						err:     errors.New("error parsing GPP header, header must have type=3"),
					},
				},
				{
					desc:  "gpp with a valid TCF2 value. Expect valid consent string and no error",
					inUri: "/setuid?gpp=DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
					expected: testOutput{
						reqInfo: gdpr.RequestInfo{
							GDPRSignal: gdpr.SignalAmbiguous,
							Consent:    "CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
						},
						err: nil,
					},
				},
				{
					desc:  "gpp does not include TCF2 string. Expect empty consent string and no error",
					inUri: "/setuid?gpp=DBABjw~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~1YNN",
					expected: testOutput{
						reqInfo: gdpr.RequestInfo{
							GDPRSignal: gdpr.SignalAmbiguous,
							Consent:    "",
						},
						err: nil,
					},
				},
			},
		},
		{
			groupDesc: "gpp_sid only",
			testCases: []aTest{
				{
					desc:  "gpp_sid is malformed, expect error",
					inUri: "/setuid?gpp_sid=malformed",
					expected: testOutput{
						reqInfo: gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous},
						err:     errors.New("Error parsing gpp_sid strconv.ParseInt: parsing \"malformed\": invalid syntax"),
					},
				},
				{
					desc:  "TCF2 found in gpp_sid list. Given that the consent string will be empty, expect an error",
					inUri: "/setuid?gpp_sid=2,6",
					expected: testOutput{
						reqInfo: gdpr.RequestInfo{GDPRSignal: gdpr.SignalYes},
						err:     nil,
					},
				},
				{
					desc:  "TCF2 not found in gpp_sid list. Expect SignalNo and no error",
					inUri: "/setuid?gpp_sid=6,8",
					expected: testOutput{
						reqInfo: gdpr.RequestInfo{GDPRSignal: gdpr.SignalNo},
						err:     nil,
					},
				},
			},
		},
		{
			groupDesc: "both gpp_sid and gpp",
			testCases: []aTest{
				{
					desc:  "TCF2 found in gpp_sid list and gpp has a valid GDPR string. Expect no error",
					inUri: "/setuid?gpp_sid=2,6&gpp=DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
					expected: testOutput{
						reqInfo: gdpr.RequestInfo{
							GDPRSignal: gdpr.SignalYes,
							Consent:    "CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
						},
						err: nil,
					},
				},
			},
		},
	}
	for _, tgroup := range testGroups {
		for _, tc := range tgroup.testCases {
			// set test
			testURL, err := url.Parse(tc.inUri)
			assert.NoError(t, err, "%s - %s", tgroup.groupDesc, tc.desc)

			query := testURL.Query()

			// run
			outReqInfo, outErr := parseGDPRFromGPP(query)

			// assertions
			assert.Equal(t, tc.expected.reqInfo, outReqInfo, "%s - %s", tgroup.groupDesc, tc.desc)
			assert.Equal(t, tc.expected.err, outErr, "%s - %s", tgroup.groupDesc, tc.desc)
		}
	}
}

func TestParseLegacyGDPRFields(t *testing.T) {
	type testInput struct {
		uri            string
		gppGDPRSignal  gdpr.Signal
		gppGDPRConsent string
	}
	type testOutput struct {
		signal  gdpr.Signal
		consent string
		err     error
	}
	testCases := []struct {
		desc     string
		in       testInput
		expected testOutput
	}{
		{
			desc: `both "gdpr" and "gdpr_consent" missing from URI, expect SignalAmbiguous, blank consent and no error`,
			in: testInput{
				uri: "/setuid?bidder=pubmatic&uid=123",
			},
			expected: testOutput{
				signal:  gdpr.SignalAmbiguous,
				consent: "",
				err:     nil,
			},
		},
		{
			desc: `invalid "gdpr" value, expect SignalAmbiguous, blank consent and error`,
			in: testInput{
				uri:           "/setuid?gdpr=2",
				gppGDPRSignal: gdpr.SignalAmbiguous,
			},
			expected: testOutput{
				signal:  gdpr.SignalAmbiguous,
				consent: "",
				err:     errors.New("the gdpr query param must be either 0 or 1. You gave 2"),
			},
		},
		{
			desc: `valid "gdpr" value but valid GDPRSignal was previously parsed before, expect SignalAmbiguous, blank consent and a warning`,
			in: testInput{
				uri:           "/setuid?gdpr=1",
				gppGDPRSignal: gdpr.SignalYes,
			},
			expected: testOutput{
				signal:  gdpr.SignalAmbiguous,
				consent: "",
				err: &errortypes.Warning{
					Message:     "'gpp_sid' signal value will be used over the one found in the deprecated 'gdpr' field.",
					WarningCode: errortypes.UnknownWarningCode,
				},
			},
		},
		{
			desc: `valid "gdpr_consent" value but valid GDPRSignal was previously parsed before, expect SignalAmbiguous, blank consent and a warning`,
			in: testInput{
				uri:            "/setuid?gdpr_consent=someConsent",
				gppGDPRConsent: "CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
			},
			expected: testOutput{
				signal:  gdpr.SignalAmbiguous,
				consent: "",
				err: &errortypes.Warning{
					Message:     "'gpp' value will be used over the one found in the deprecated 'gdpr_consent' field.",
					WarningCode: errortypes.UnknownWarningCode,
				},
			},
		},
	}
	for _, tc := range testCases {
		// set test
		testURL, err := url.Parse(tc.in.uri)
		assert.NoError(t, err, tc.desc)

		query := testURL.Query()

		// run
		outSignal, outConsent, outErr := parseLegacyGDPRFields(query, tc.in.gppGDPRSignal, tc.in.gppGDPRConsent)

		// assertions
		assert.Equal(t, tc.expected.signal, outSignal, tc.desc)
		assert.Equal(t, tc.expected.consent, outConsent, tc.desc)
		assert.Equal(t, tc.expected.err, outErr, tc.desc)
	}
}

func TestExtractGDPRInfo(t *testing.T) {
	type testOutput struct {
		requestInfo gdpr.RequestInfo
		err         error
	}
	type testCase struct {
		desc     string
		inUri    string
		expected testOutput
	}
	testSuite := []struct {
		sDesc string
		tests []testCase
	}{
		{
			sDesc: "no gdpr nor gpp values in query",
			tests: []testCase{
				{
					desc:  "expect blank consent, signalNo and nil error",
					inUri: "/setuid?bidder=pubmatic&uid=123",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{
							Consent:    "",
							GDPRSignal: gdpr.SignalAmbiguous,
						},
						err: nil,
					},
				},
			},
		},
		{
			sDesc: "missing gpp, gdpr only",
			tests: []testCase{
				{
					desc:  "Invalid gdpr signal value in query, expect blank request info and error",
					inUri: "/setuid?gdpr=2",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous},
						err:         errors.New("the gdpr query param must be either 0 or 1. You gave 2"),
					},
				},
				{
					desc:  "GDPR equals 0, blank consent, expect blank consent, signalNo and nil error",
					inUri: "/setuid?gdpr=0",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{GDPRSignal: gdpr.SignalNo},
						err:         nil,
					},
				},
				{
					desc:  "GDPR equals 1, blank consent, expect blank request info and error",
					inUri: "/setuid?gdpr=1",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous},
						err:         errors.New("GDPR consent is required when gdpr signal equals 1"),
					},
				},
				{
					desc:  "GDPR equals 0, non-blank consent, expect non-blank request info and nil error",
					inUri: "/setuid?gdpr=0&gdpr_consent=someConsent",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{
							Consent:    "someConsent",
							GDPRSignal: gdpr.SignalNo,
						},
						err: nil,
					},
				},
				{
					desc:  "GDPR equals 1, non-blank consent, expect non-blank request info and nil error",
					inUri: "/setuid?gdpr=1&gdpr_consent=someConsent",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{
							Consent:    "someConsent",
							GDPRSignal: gdpr.SignalYes,
						},
						err: nil,
					},
				},
			},
		},
		{
			sDesc: "missing gdpr, gpp only",
			tests: []testCase{
				{
					desc:  "Malformed GPP_SID string, expect blank request info and error",
					inUri: "/setuid?gpp_sid=malformed",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous},
						err:         errors.New("Error parsing gpp_sid strconv.ParseInt: parsing \"malformed\": invalid syntax"),
					},
				},
				{
					desc:  "Valid GPP_SID string but invalid GPP string in query, expect blank request info and error",
					inUri: "/setuid?gpp=malformed&gpp_sid=2",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous},
						err:         errors.New("error parsing GPP header, header must have type=3"),
					},
				},
				{
					desc:  "SectionTCFEU2 not found in GPP string, expect blank consent and signalAmbiguous",
					inUri: "/setuid?gpp=DBABBgA~xlgWEYCZAA",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{
							Consent:    "",
							GDPRSignal: gdpr.SignalAmbiguous,
						},
						err: nil,
					},
				},
				{
					desc:  "No GPP string, nor SectionTCFEU2 found in SID list in query, expect blank consent and signalAmbiguous",
					inUri: "/setuid?gpp_sid=3,6",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{
							Consent:    "",
							GDPRSignal: gdpr.SignalNo,
						},
						err: nil,
					},
				},
				{
					desc:  "No GPP string, SectionTCFEU2 found in SID list in query, expect blank request info and error",
					inUri: "/setuid?gpp_sid=2",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous},
						err:         errors.New("GDPR consent is required when gdpr signal equals 1"),
					},
				},
				{
					desc:  "SectionTCFEU2 only found in SID list, expect blank request info and error",
					inUri: "/setuid?gpp=DBABBgA~xlgWEYCZAA&gpp_sid=2",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous},
						err:         errors.New("GDPR consent is required when gdpr signal equals 1"),
					},
				},
				{
					desc:  "SectionTCFEU2 found in GPP string but SID list is nil, expect valid consent and SignalAmbiguous",
					inUri: "/setuid?gpp=DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{
							Consent:    "CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
							GDPRSignal: gdpr.SignalAmbiguous,
						},
						err: nil,
					},
				},
				{
					desc:  "SectionTCFEU2 found in GPP string but not in the non-nil SID list, expect valid consent and signalNo",
					inUri: "/setuid?gpp=DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA&gpp_sid=6",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{
							Consent:    "CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
							GDPRSignal: gdpr.SignalNo,
						},
						err: nil,
					},
				},
				{
					desc:  "SectionTCFEU2 found both in GPP string and SID list, expect valid consent and signalYes",
					inUri: "/setuid?gpp=DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA&gpp_sid=2,4",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{
							Consent:    "CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
							GDPRSignal: gdpr.SignalYes,
						},
						err: nil,
					},
				},
			},
		},
		{
			sDesc: "GPP values take priority over GDPR",
			tests: []testCase{
				{
					desc:  "SignalNo in gdpr field but SignalYes in SID list, CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA consent in gpp but legacyConsent in gdpr_consent, expect GPP values to prevail",
					inUri: "/setuid?gpp=DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA&gpp_sid=2,4&gdpr=0&gdpr_consent=legacyConsent",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{
							Consent:    "CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
							GDPRSignal: gdpr.SignalYes,
						},
						err: &errortypes.Warning{
							Message:     "'gpp' value will be used over the one found in the deprecated 'gdpr_consent' field.",
							WarningCode: errortypes.UnknownWarningCode,
						},
					},
				},
				{
					desc:  "SignalNo in gdpr field but SignalYes in SID list because SectionTCFEU2 is listed, expect GPP to prevail",
					inUri: "/setuid?gpp=DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA&gpp_sid=2,4&gdpr=0",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{
							Consent:    "CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
							GDPRSignal: gdpr.SignalYes,
						},
						err: &errortypes.Warning{
							Message:     "'gpp_sid' signal value will be used over the one found in the deprecated 'gdpr' field.",
							WarningCode: errortypes.UnknownWarningCode,
						},
					},
				},
				{
					desc:  "No gpp string in URL query, use gdpr_consent and SignalYes found in SID list because SectionTCFEU2 is listed",
					inUri: "/setuid?gpp_sid=2,4&gdpr_consent=legacyConsent",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{
							Consent:    "",
							GDPRSignal: gdpr.SignalAmbiguous,
						},
						err: errors.New("GDPR consent is required when gdpr signal equals 1"),
					},
				},
				{
					desc:  "SectionTCFEU2 not found in GPP string but found in SID list, choose the GDPR_CONSENT and GPP_SID signal",
					inUri: "/setuid?gpp=DBABBgA~xlgWEYCZAA&gpp_sid=2&gdpr=0&gdpr_consent=legacyConsent",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{
							Consent:    "",
							GDPRSignal: gdpr.SignalAmbiguous,
						},
						err: errors.New("GDPR consent is required when gdpr signal equals 1"),
					},
				},
				{
					desc:  "SectionTCFEU2 found in GPP string but not in SID list, choose GDPR signal GPP consent value",
					inUri: "/setuid?gpp=DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA&gpp_sid=6&gdpr=1&gdpr_consent=legacyConsent",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{
							Consent:    "CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA",
							GDPRSignal: gdpr.SignalNo,
						},
						err: &errortypes.Warning{
							Message:     "'gpp' value will be used over the one found in the deprecated 'gdpr_consent' field.",
							WarningCode: errortypes.UnknownWarningCode,
						},
					},
				},
				{
					desc:  "SectionTCFEU2 not found in GPP, use GDPR_CONSENT value. SignalYes found in gdpr field, but not in the valid SID list, expect SignalNo",
					inUri: "/setuid?gpp=DBABBgA~xlgWEYCZAA&gpp_sid=6&gdpr=1&gdpr_consent=legacyConsent",
					expected: testOutput{
						requestInfo: gdpr.RequestInfo{
							Consent:    "",
							GDPRSignal: gdpr.SignalNo,
						},
						err: &errortypes.Warning{
							Message:     "'gpp_sid' signal value will be used over the one found in the deprecated 'gdpr' field.",
							WarningCode: errortypes.UnknownWarningCode,
						},
					},
				},
			},
		},
	}

	for _, ts := range testSuite {
		for _, tc := range ts.tests {
			// set test
			testURL, err := url.Parse(tc.inUri)
			assert.NoError(t, err, tc.desc)

			query := testURL.Query()

			// run
			outReqInfo, outErr := extractGDPRInfo(query)

			// assertions
			assert.Equal(t, tc.expected.requestInfo, outReqInfo, tc.desc)
			assert.Equal(t, tc.expected.err, outErr, tc.desc)
		}
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
		expectedAnalytics      func(*MockAnalyticsRunner)
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
			expectedAnalytics: func(a *MockAnalyticsRunner) {
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
			expectedAnalytics: func(a *MockAnalyticsRunner) {
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
			expectedAnalytics: func(a *MockAnalyticsRunner) {
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
			expectedAnalytics: func(a *MockAnalyticsRunner) {
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
			expectedAnalytics: func(a *MockAnalyticsRunner) {
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
			expectedAnalytics: func(a *MockAnalyticsRunner) {
				expected := analytics.SetUIDObject{
					Status:  400,
					Bidder:  "pubmatic",
					UID:     "",
					Errors:  []error{errors.New("GDPR consent is required when gdpr signal equals 1")},
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
			expectedAnalytics: func(a *MockAnalyticsRunner) {
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
			expectedAnalytics: func(a *MockAnalyticsRunner) {
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
			expectedAnalytics: func(a *MockAnalyticsRunner) {
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
				m.On("RecordSetUid", metrics.SetUidAccountConfigMalformed).Once()
			},
			expectedAnalytics: func(a *MockAnalyticsRunner) {
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
	}

	for _, test := range testCases {
		analyticsEngine := &MockAnalyticsRunner{}
		test.expectedAnalytics(analyticsEngine)

		metricsEngine := &metrics.MetricsEngineMock{}
		test.expectedMetrics(metricsEngine)

		req := httptest.NewRequest("GET", test.uri, nil)
		for _, v := range test.cookies {
			addCookie(req, v)
		}
		response := doRequest(req, analyticsEngine, metricsEngine, test.syncersBidderNameToKey, test.gdprAllowsHostCookies, false, false, test.cfgAccountRequired, 0, nil, "")

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
	analytics := analyticsBuild.New(&config.Analytics{})
	metrics := &metricsConf.NilMetricsEngine{}
	response := doRequest(request, analytics, metrics, syncersBidderNameToKey, true, false, false, false, 0, nil, "")

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
		{
			urlValues:      url.Values{"f": []string{""}},
			syncer:         fakeSyncer{key: "a", defaultSyncType: usersync.SyncTypeRedirect},
			expectedFormat: "i",
			description:    "parameter given is empty (by empty item), use default sync type redirect",
		},
		{
			urlValues:      url.Values{"f": []string{}},
			syncer:         fakeSyncer{key: "a", formatOverride: "i"},
			expectedFormat: "i",
			description:    "format not provided, but formatOverride is defined, expect i",
		},
		{
			urlValues:      url.Values{"f": []string{}},
			syncer:         fakeSyncer{key: "a", formatOverride: "b"},
			expectedFormat: "b",
			description:    "format not provided, but formatOverride is defined, expect b",
		},
		{
			urlValues:      url.Values{"f": []string{}},
			syncer:         fakeSyncer{key: "a", formatOverride: "b", defaultSyncType: usersync.SyncTypeRedirect},
			expectedFormat: "b",
			description:    "format not provided, default is defined but formatOverride is defined as well, expect b",
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

func TestIsSyncerPriority(t *testing.T) {
	testCases := []struct {
		name                           string
		givenBidderNameFromSyncerQuery string
		givenPriorityGroups            [][]string
		expected                       bool
	}{
		{
			name:                           "priority-tier-1",
			givenBidderNameFromSyncerQuery: "a",
			givenPriorityGroups:            [][]string{{"a"}},
			expected:                       true,
		},
		{
			name:                           "priority-tier-other",
			givenBidderNameFromSyncerQuery: "c",
			givenPriorityGroups:            [][]string{{"a"}, {"b", "c"}},
			expected:                       true,
		},
		{
			name:                           "priority-case-insensitive",
			givenBidderNameFromSyncerQuery: "A",
			givenPriorityGroups:            [][]string{{"a"}},
			expected:                       true,
		},
		{
			name:                           "not-priority-empty",
			givenBidderNameFromSyncerQuery: "a",
			givenPriorityGroups:            [][]string{},
			expected:                       false,
		},
		{
			name:                           "not-priority-not-defined",
			givenBidderNameFromSyncerQuery: "a",
			givenPriorityGroups:            [][]string{{"b"}},
			expected:                       false,
		},
		{
			name:                           "no-bidder",
			givenBidderNameFromSyncerQuery: "",
			givenPriorityGroups:            [][]string{{"b"}},
			expected:                       false,
		},
		{
			name:                           "no-priority-groups",
			givenBidderNameFromSyncerQuery: "a",
			givenPriorityGroups:            [][]string{},
			expected:                       false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			isPriority := isSyncerPriority(test.givenBidderNameFromSyncerQuery, test.givenPriorityGroups)
			assert.Equal(t, test.expected, isPriority)
		})
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
			pbsCookie.Sync(key, value)
		}
		addCookie(request, pbsCookie)
	}
	return request
}

func doRequest(req *http.Request, analytics analytics.Runner, metrics metrics.MetricsEngine, syncersBidderNameToKey map[string]string, gdprAllowsHostCookies, gdprReturnsError, gdprReturnsMalformedError, cfgAccountRequired bool, maxCookieSize int, priorityGroups [][]string, formatOverride string) *httptest.ResponseRecorder {
	cfg := config.Configuration{
		AccountRequired: cfgAccountRequired,
		AccountDefaults: config.Account{},
		UserSync: config.UserSync{
			PriorityGroups: priorityGroups,
		},
		HostCookie: config.HostCookie{
			MaxCookieSizeBytes: maxCookieSize,
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
		syncersByBidder[bidderName] = fakeSyncer{key: syncerKey, defaultSyncType: usersync.SyncTypeIFrame, formatOverride: formatOverride}
		if priorityGroups == nil {
			cfg.UserSync.PriorityGroups = [][]string{{}}
			cfg.UserSync.PriorityGroups[0] = append(cfg.UserSync.PriorityGroups[0], bidderName)
		}
	}

	fakeAccountsFetcher := FakeAccountsFetcher{AccountData: map[string]json.RawMessage{
		"valid_acct":        json.RawMessage(`{"disabled":false}`),
		"disabled_acct":     json.RawMessage(`{"disabled":true}`),
		"malformed_acct":    json.RawMessage(`{"disabled":"malformed"}`),
		"invalid_json_acct": json.RawMessage(`{"}`),

		"valid_acct_with_valid_activities_usersync_enabled":  json.RawMessage(`{"privacy":{"allowactivities":{"syncUser":{"default": true}}}}`),
		"valid_acct_with_valid_activities_usersync_disabled": json.RawMessage(`{"privacy":{"allowactivities":{"syncUser":{"default": false}}}}`),
		"valid_acct_with_invalid_activities":                 json.RawMessage(`{"privacy":{"allowactivities":{"syncUser":{"rules":[{"condition":{"componentName": ["bidderA.bidderB.bidderC"]}}]}}}}`),
	}}

	endpoint := NewSetUIDEndpoint(&cfg, syncersByBidder, gdprPermsBuilder, tcf2ConfigBuilder, analytics, fakeAccountsFetcher, metrics)
	response := httptest.NewRecorder()
	endpoint(response, req, nil)
	return response
}

func addCookie(req *http.Request, cookie *usersync.Cookie) {
	httpCookie, _ := ToHTTPCookie(cookie)
	req.AddCookie(httpCookie)
}

func parseCookieString(t *testing.T, response *httptest.ResponseRecorder) *usersync.Cookie {
	decoder := usersync.Base64Decoder{}
	cookieString := response.Header().Get("Set-Cookie")
	parser := regexp.MustCompile("uids=(.*?);")
	res := parser.FindStringSubmatch(cookieString)
	assert.Equal(t, 2, len(res))
	httpCookie := http.Cookie{
		Name:  "uids",
		Value: res[1],
	}
	return decoder.Decode(httpCookie.Value)
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

func (g *fakePermsSetUID) AuctionActivitiesAllowed(ctx context.Context, bidderCoreName openrtb_ext.BidderName, bidder openrtb_ext.BidderName) gdpr.AuctionPermissions {
	return gdpr.AuctionPermissions{
		AllowBidRequest: g.personalInfoAllowed,
		PassGeo:         g.personalInfoAllowed,
		PassID:          g.personalInfoAllowed,
	}
}

type fakeSyncer struct {
	key             string
	defaultSyncType usersync.SyncType
	formatOverride  string
}

func (s fakeSyncer) Key() string {
	return s.key
}

func (s fakeSyncer) DefaultResponseFormat() usersync.SyncType {
	switch s.formatOverride {
	case "b":
		return usersync.SyncTypeIFrame
	case "i":
		return usersync.SyncTypeRedirect
	default:
		return s.defaultSyncType
	}
}

func (s fakeSyncer) SupportsType(syncTypes []usersync.SyncType) bool {
	return true
}

func (s fakeSyncer) GetSync(syncTypes []usersync.SyncType, privacyMacros macros.UserSyncPrivacy) (usersync.Sync, error) {
	return usersync.Sync{}, nil
}

func ToHTTPCookie(cookie *usersync.Cookie) (*http.Cookie, error) {
	encoder := usersync.Base64Encoder{}
	encodedCookie, err := encoder.Encode(cookie)
	if err != nil {
		return nil, nil
	}

	return &http.Cookie{
		Name:    uidCookieName,
		Value:   encodedCookie,
		Expires: time.Now().Add((90 * 24 * time.Hour)),
		Path:    "/",
	}, nil
}

func getUIDFromHeader(setCookieHeader string) string {
	cookies := strings.Split(setCookieHeader, ";")
	for _, cookie := range cookies {
		trimmedCookie := strings.TrimSpace(cookie)
		if strings.HasPrefix(trimmedCookie, "uids=") {
			parts := strings.SplitN(trimmedCookie, "=", 2)
			if len(parts) == 2 {
				return parts[1]
			}
		}
	}
	return ""
}
