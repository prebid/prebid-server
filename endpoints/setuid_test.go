package endpoints

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/usersync"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/openrtb_ext"

	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	metricsConf "github.com/prebid/prebid-server/metrics/config"
)

func TestSetUIDEndpoint(t *testing.T) {
	testCases := []struct {
		uri                   string
		validFamilyNames      []string
		existingSyncs         map[string]string
		gdprAllowsHostCookies bool
		gdprReturnsError      bool
		expectedSyncs         map[string]string
		expectedRespMessage   string
		expectedResponseCode  int
		description           string
	}{
		{
			uri:                   "/setuid?bidder=pubmatic&uid=123",
			validFamilyNames:      []string{"pubmatic"},
			existingSyncs:         nil,
			gdprAllowsHostCookies: true,
			expectedSyncs:         map[string]string{"pubmatic": "123"},
			expectedResponseCode:  http.StatusOK,
			description:           "Set uid for valid bidder",
		},
		{
			uri:                   "/setuid?bidder=unsupported-bidder&uid=123",
			validFamilyNames:      []string{},
			existingSyncs:         nil,
			gdprAllowsHostCookies: true,
			expectedSyncs:         nil,
			expectedResponseCode:  http.StatusBadRequest,
			description:           "Don't set uid for an unsupported bidder",
		},
		{
			uri:                   "/setuid?bidder=&uid=123",
			validFamilyNames:      []string{"pubmatic"},
			existingSyncs:         nil,
			gdprAllowsHostCookies: true,
			expectedSyncs:         nil,
			expectedResponseCode:  http.StatusBadRequest,
			description:           "Don't set uid for an empty bidder",
		},
		{
			uri:                   "/setuid?bidder=unsupported-bidder&uid=123",
			validFamilyNames:      []string{},
			existingSyncs:         map[string]string{"pubmatic": "1234"},
			gdprAllowsHostCookies: true,
			expectedSyncs:         nil,
			expectedResponseCode:  http.StatusBadRequest,
			description: "No need to set existing syncs back in response for a request " +
				"to set uid for an unsupported bidder",
		},
		{
			uri:                   "/setuid?bidder=&uid=123",
			validFamilyNames:      []string{"pubmatic"},
			existingSyncs:         map[string]string{"pubmatic": "1234"},
			gdprAllowsHostCookies: true,
			expectedSyncs:         nil,
			expectedResponseCode:  http.StatusBadRequest,
			description: "No need to set existing syncs back in response for a request " +
				"to set uid for an empty bidder",
		},
		{
			uri:                   "/setuid?bidder=pubmatic",
			validFamilyNames:      []string{"pubmatic"},
			existingSyncs:         map[string]string{"pubmatic": "1234"},
			gdprAllowsHostCookies: true,
			expectedSyncs:         map[string]string{},
			expectedResponseCode:  http.StatusOK,
			description:           "Unset uid for a bidder if the request contains an empty uid for that bidder",
		},
		{
			uri:                   "/setuid?bidder=pubmatic&uid=123",
			validFamilyNames:      []string{"pubmatic"},
			existingSyncs:         map[string]string{"rubicon": "def"},
			gdprAllowsHostCookies: true,
			expectedSyncs:         map[string]string{"pubmatic": "123", "rubicon": "def"},
			expectedResponseCode:  http.StatusOK,
			description:           "Add the uid for the requested bidder to the list of existing syncs",
		},
		{
			uri:                   "/setuid?bidder=pubmatic&uid=123&gdpr=0",
			validFamilyNames:      []string{"pubmatic"},
			existingSyncs:         nil,
			gdprAllowsHostCookies: true,
			expectedSyncs:         map[string]string{"pubmatic": "123"},
			expectedResponseCode:  http.StatusOK,
			description:           "Don't care about GDPR consent if GDPR is set to 0",
		},
		{
			uri:                  "/setuid?bidder=pubmatic&uid=123",
			validFamilyNames:     []string{"pubmatic"},
			existingSyncs:        nil,
			expectedSyncs:        nil,
			expectedResponseCode: http.StatusOK,
			expectedRespMessage:  "The gdpr_consent string prevents cookies from being saved",
			description:          "Return err message if the GDPR consent doesn't allow syncs for the given bidder",
		},
		{
			uri:                   "/setuid?uid=123",
			validFamilyNames:      []string{"appnexus"},
			existingSyncs:         nil,
			expectedSyncs:         nil,
			gdprAllowsHostCookies: true,
			expectedResponseCode:  http.StatusBadRequest,
			expectedRespMessage:   `"bidder" query param is required`,
			description:           "Return an error if the bidder param is missing from the request",
		},
		{
			uri:                   "/setuid?bidder=appnexus&uid=123&gdpr=2",
			validFamilyNames:      []string{"appnexus"},
			existingSyncs:         nil,
			expectedSyncs:         nil,
			gdprAllowsHostCookies: true,
			expectedResponseCode:  http.StatusBadRequest,
			expectedRespMessage:   "the gdpr query param must be either 0 or 1. You gave 2",
			description:           "Return an error if GDPR is set to anything else other that 0 or 1",
		},
		{
			uri:                   "/setuid?bidder=appnexus&uid=123&gdpr=1",
			validFamilyNames:      []string{"appnexus"},
			existingSyncs:         nil,
			expectedSyncs:         nil,
			gdprAllowsHostCookies: true,
			expectedResponseCode:  http.StatusBadRequest,
			expectedRespMessage:   "gdpr_consent is required when gdpr=1",
			description:           "Return an error if GDPR is set to 1 but GDPR consent string is missing",
		},
		{
			uri: "/setuid?bidder=pubmatic&uid=123&gdpr_consent=" +
				"BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw",
			validFamilyNames:     []string{"pubmatic"},
			existingSyncs:        nil,
			expectedSyncs:        nil,
			gdprReturnsError:     true,
			expectedResponseCode: http.StatusBadRequest,
			expectedRespMessage: "No global vendor list was available to interpret this consent string. " +
				"If this is a new, valid version, it should become available soon.",
			description: "Return an error if the GDPR string is either malformed or using a newer version that isn't yet supported",
		},
		{
			uri: "/setuid?bidder=pubmatic&uid=123&gdpr=1&gdpr_consent=" +
				"BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw",
			validFamilyNames:     []string{"pubmatic"},
			existingSyncs:        nil,
			expectedSyncs:        nil,
			expectedResponseCode: http.StatusOK,
			expectedRespMessage:  "The gdpr_consent string prevents cookies from being saved",
			description:          "Shouldn't set uid for a bidder if it is not allowed by the GDPR consent string",
		},
		{
			uri: "/setuid?bidder=pubmatic&uid=123&gdpr=1&gdpr_consent=" +
				"BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw",
			validFamilyNames:      []string{"pubmatic"},
			gdprAllowsHostCookies: true,
			existingSyncs:         nil,
			expectedSyncs:         map[string]string{"pubmatic": "123"},
			expectedResponseCode:  http.StatusOK,
			description:           "Should set uid for a bidder that is allowed by the GDPR consent string",
		},
	}

	metrics := &metricsConf.DummyMetricsEngine{}
	for _, test := range testCases {
		response := doRequest(makeRequest(test.uri, test.existingSyncs), metrics,
			test.validFamilyNames, test.gdprAllowsHostCookies, test.gdprReturnsError)
		assert.Equal(t, test.expectedResponseCode, response.Code, "Test Case: %s. /setuid returned unexpected error code", test.description)

		if test.expectedSyncs != nil {
			assertHasSyncs(t, test.description, response, test.expectedSyncs)
		} else {
			assert.Equal(t, "", response.Header().Get("Set-Cookie"), "Test Case: %s. /setuid returned unexpected cookie", test.description)
		}

		if test.expectedRespMessage != "" {
			assert.Equal(t, test.expectedRespMessage, response.Body.String(), "Test Case: %s. /setuid returned unexpected message")
		}
	}
}

func TestSetUIDEndpointMetrics(t *testing.T) {
	testCases := []struct {
		uri                   string
		cookies               []*usersync.PBSCookie
		validFamilyNames      []string
		gdprAllowsHostCookies bool
		expectedMetricAction  metrics.RequestAction
		expectedMetricBidder  openrtb_ext.BidderName
		expectedResponseCode  int
		description           string
	}{
		{
			uri:                   "/setuid?bidder=pubmatic&uid=123",
			cookies:               []*usersync.PBSCookie{},
			validFamilyNames:      []string{"pubmatic"},
			gdprAllowsHostCookies: true,
			expectedMetricAction:  metrics.RequestActionSet,
			expectedMetricBidder:  openrtb_ext.BidderName("pubmatic"),
			expectedResponseCode:  200,
			description:           "Success - Sync",
		},
		{
			uri:                   "/setuid?bidder=pubmatic&uid=",
			cookies:               []*usersync.PBSCookie{},
			validFamilyNames:      []string{"pubmatic"},
			gdprAllowsHostCookies: true,
			expectedMetricAction:  metrics.RequestActionSet,
			expectedMetricBidder:  openrtb_ext.BidderName("pubmatic"),
			expectedResponseCode:  200,
			description:           "Success - Unsync",
		},
		{
			uri:                   "/setuid?bidder=pubmatic&uid=123",
			cookies:               []*usersync.PBSCookie{usersync.NewPBSCookieWithOptOut()},
			validFamilyNames:      []string{"pubmatic"},
			gdprAllowsHostCookies: true,
			expectedMetricAction:  metrics.RequestActionOptOut,
			expectedResponseCode:  401,
			description:           "Cookie Opted Out",
		},
		{
			uri:                   "/setuid?bidder=pubmatic&uid=123",
			cookies:               []*usersync.PBSCookie{},
			validFamilyNames:      []string{},
			gdprAllowsHostCookies: true,
			expectedMetricAction:  metrics.RequestActionErr,
			expectedResponseCode:  400,
			description:           "Unsupported Cookie Name",
		},
		{
			uri:                   "/setuid?bidder=pubmatic&uid=123&gdpr=1",
			cookies:               []*usersync.PBSCookie{},
			validFamilyNames:      []string{"pubmatic"},
			gdprAllowsHostCookies: false,
			expectedMetricAction:  metrics.RequestActionGDPR,
			expectedMetricBidder:  openrtb_ext.BidderName("pubmatic"),
			expectedResponseCode:  400,
			description:           "Prevented By GDPR",
		},
	}

	for _, test := range testCases {
		metricsEngine := &metrics.MetricsEngineMock{}
		expectedLabels := metrics.UserLabels{
			Action: test.expectedMetricAction,
			Bidder: test.expectedMetricBidder,
		}
		metricsEngine.On("RecordUserIDSet", expectedLabels).Once()

		req := httptest.NewRequest("GET", test.uri, nil)
		for _, v := range test.cookies {
			addCookie(req, v)
		}
		response := doRequest(req, metricsEngine, test.validFamilyNames, test.gdprAllowsHostCookies, false)

		assert.Equal(t, test.expectedResponseCode, response.Code, test.description)
		metricsEngine.AssertExpectations(t)
	}
}

func TestOptedOut(t *testing.T) {
	request := httptest.NewRequest("GET", "/setuid?bidder=pubmatic&uid=123", nil)
	cookie := usersync.NewPBSCookie()
	cookie.SetPreference(false)
	addCookie(request, cookie)
	validFamilyNames := []string{"pubmatic"}
	metrics := &metricsConf.DummyMetricsEngine{}
	response := doRequest(request, metrics, validFamilyNames, true, false)

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

func TestGetFamilyName(t *testing.T) {
	testCases := []struct {
		urlValues     url.Values
		expectedName  string
		expectedError string
		description   string
	}{
		{
			urlValues:    url.Values{"bidder": []string{"valid"}},
			expectedName: "valid",
			description:  "Should return no error for valid family name",
		},
		{
			urlValues:     url.Values{"bidder": []string{"VALID"}},
			expectedError: "The bidder name provided is not supported by Prebid Server",
			description:   "Should return error for different case",
		},
		{
			urlValues:     url.Values{"bidder": []string{"invalid"}},
			expectedError: "The bidder name provided is not supported by Prebid Server",
			description:   "Should return an error for unsupported bidder",
		},
		{
			urlValues:     url.Values{"bidder": []string{}},
			expectedError: `"bidder" query param is required`,
			description:   "Should return an error for empty bidder name",
		},
		{
			urlValues:     url.Values{},
			expectedError: `"bidder" query param is required`,
			description:   "Should return an error for missing bidder name",
		},
	}

	for _, test := range testCases {

		name, err := getFamilyName(test.urlValues, map[string]struct{}{"valid": {}})

		assert.Equal(t, test.expectedName, name, test.description)

		if test.expectedError != "" {
			assert.EqualError(t, err, test.expectedError, test.description)
		} else {
			assert.NoError(t, err, test.description)
		}
	}
}

func assertHasSyncs(t *testing.T, testCase string, resp *httptest.ResponseRecorder, syncs map[string]string) {
	t.Helper()
	cookie := parseCookieString(t, resp)
	assert.Equal(t, len(syncs), cookie.LiveSyncCount(), "Test Case: %s. /setuid response doesn't contain expected number of syncs", testCase)
	for bidder, uid := range syncs {
		assert.True(t, cookie.HasLiveSync(bidder), "Test Case: %s. /setuid response cookie doesn't contain uid for bidder: %s", testCase, bidder)
		actualUID, _, _ := cookie.GetUID(bidder)
		assert.Equal(t, uid, actualUID, "Test Case: %s. /setuid response cookie doesn't contain correct uid for bidder: %s", testCase, bidder)
	}
}

func makeRequest(uri string, existingSyncs map[string]string) *http.Request {
	request := httptest.NewRequest("GET", uri, nil)
	if len(existingSyncs) > 0 {
		pbsCookie := usersync.NewPBSCookie()
		for family, value := range existingSyncs {
			pbsCookie.TrySync(family, value)
		}
		addCookie(request, pbsCookie)
	}
	return request
}

func doRequest(req *http.Request, metrics metrics.MetricsEngine, validFamilyNames []string, gdprAllowsHostCookies bool, gdprReturnsError bool) *httptest.ResponseRecorder {
	cfg := config.Configuration{}
	perms := &mockPermsSetUID{
		allowHost:           gdprAllowsHostCookies,
		errorHost:           gdprReturnsError,
		personalInfoAllowed: true,
	}
	analytics := analyticsConf.NewPBSAnalytics(&cfg.Analytics)
	syncers := make(map[openrtb_ext.BidderName]usersync.Usersyncer)
	for _, name := range validFamilyNames {
		syncers[openrtb_ext.BidderName(name)] = newFakeSyncer(name)
	}

	endpoint := NewSetUIDEndpoint(cfg.HostCookie, syncers, perms, analytics, metrics)
	response := httptest.NewRecorder()
	endpoint(response, req, nil)
	return response
}

func addCookie(req *http.Request, cookie *usersync.PBSCookie) {
	req.AddCookie(cookie.ToHTTPCookie(time.Duration(1) * time.Hour))
}

func parseCookieString(t *testing.T, response *httptest.ResponseRecorder) *usersync.PBSCookie {
	cookieString := response.Header().Get("Set-Cookie")
	parser := regexp.MustCompile("uids=(.*?);")
	res := parser.FindStringSubmatch(cookieString)
	assert.Equal(t, 2, len(res))
	httpCookie := http.Cookie{
		Name:  "uids",
		Value: res[1],
	}
	return usersync.ParsePBSCookie(&httpCookie)
}

type mockPermsSetUID struct {
	allowHost           bool
	errorHost           bool
	personalInfoAllowed bool
}

func (g *mockPermsSetUID) HostCookiesAllowed(ctx context.Context, gdprSignal gdpr.Signal, consent string) (bool, error) {
	var err error
	if g.errorHost {
		err = errors.New("something went wrong")
	}
	return g.allowHost, err
}

func (g *mockPermsSetUID) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, gdprSignal gdpr.Signal, consent string) (bool, error) {
	return false, nil
}

func (g *mockPermsSetUID) AuctionActivitiesAllowed(ctx context.Context, bidder openrtb_ext.BidderName, PublisherID string, gdprSignal gdpr.Signal, consent string, weakVendorEnforcement bool) (allowBidRequest bool, passGeo bool, passID bool, err error) {
	return g.personalInfoAllowed, g.personalInfoAllowed, g.personalInfoAllowed, nil
}

func newFakeSyncer(familyName string) usersync.Usersyncer {
	return fakeSyncer{
		familyName: familyName,
	}
}

type fakeSyncer struct {
	familyName string
}

// FamilyNames implements the Usersyncer interface.
func (s fakeSyncer) FamilyName() string {
	return s.familyName
}

// GetUsersyncInfo implements the Usersyncer interface with a no-op.
func (s fakeSyncer) GetUsersyncInfo(privacyPolicies privacy.Policies) (*usersync.UsersyncInfo, error) {
	return nil, nil
}
