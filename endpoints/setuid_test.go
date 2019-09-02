package endpoints

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/prebid/prebid-server/usersync"

	"github.com/prebid/prebid-server/openrtb_ext"

	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	metricsConf "github.com/prebid/prebid-server/pbsmetrics/config"
)

func TestNormalSet(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123", nil), true, false)
	assertIntsMatch(t, http.StatusOK, response.Code)
	assertHasSyncs(t, response, map[string]string{
		"pubmatic": "123",
	})
}

func TestUnset(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic", map[string]string{"pubmatic": "1234"}), true, false)
	assertIntsMatch(t, http.StatusOK, response.Code)
	assertHasSyncs(t, response, nil)
}

func TestMergeSet(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123", map[string]string{"rubicon": "def"}), true, false)
	assertIntsMatch(t, http.StatusOK, response.Code)
	assertHasSyncs(t, response, map[string]string{
		"pubmatic": "123",
		"rubicon":  "def",
	})
}

func TestGDPRPrevention(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123", nil), false, false)
	assertIntsMatch(t, http.StatusOK, response.Code)
	assertStringsMatch(t, "The gdpr_consent string prevents cookies from being saved", response.Body.String())
	assertNoCookie(t, response)
}

func TestGDPRConsentError(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123&gdpr_consent=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw", nil), false, true)
	assertIntsMatch(t, http.StatusBadRequest, response.Code)
	assertStringsMatch(t, "No global vendor list was available to interpret this consent string. If this is a new, valid version, it should become available soon.", response.Body.String())
	assertNoCookie(t, response)
}

func TestInapplicableGDPR(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123&gdpr=0", nil), false, false)
	assertIntsMatch(t, http.StatusOK, response.Code)
	assertHasSyncs(t, response, map[string]string{
		"pubmatic": "123",
	})
}

func TestExplicitGDPRPrevention(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123&gdpr=1&gdpr_consent=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw", nil), false, false)
	assertIntsMatch(t, http.StatusOK, response.Code)
	assertStringsMatch(t, "The gdpr_consent string prevents cookies from being saved", response.Body.String())
	assertNoCookie(t, response)
}

func assertNoCookie(t *testing.T, resp *httptest.ResponseRecorder) {
	t.Helper()
	assertStringsMatch(t, "", resp.Header().Get("Set-Cookie"))
}

func TestBadRequests(t *testing.T) {
	assertBadRequest(t, "/setuid?uid=123", `"bidder" query param is required`)
	assertBadRequest(t, "/setuid?bidder=appnexus&uid=123&gdpr=2", "the gdpr query param must be either 0 or 1. You gave 2")
	assertBadRequest(t, "/setuid?bidder=appnexus&uid=123&gdpr=1", "gdpr_consent is required when gdpr=1")
}

func TestOptedOut(t *testing.T) {
	request := httptest.NewRequest("GET", "/setuid?bidder=pubmatic&uid=123", nil)
	cookie := usersync.NewPBSCookie()
	cookie.SetPreference(false)
	addCookie(request, cookie)
	response := doRequest(request, true, false)

	assertIntsMatch(t, http.StatusUnauthorized, response.Code)
}

func assertHasSyncs(t *testing.T, resp *httptest.ResponseRecorder, syncs map[string]string) {
	t.Helper()
	cookie := parseCookieString(t, resp)
	assertIntsMatch(t, len(syncs), cookie.LiveSyncCount())
	for bidder, value := range syncs {
		assertBoolsMatch(t, true, cookie.HasLiveSync(bidder))
		assertSyncValue(t, cookie, bidder, value)
	}
}

func assertBadRequest(t *testing.T, uri string, errMsg string) {
	t.Helper()
	response := doRequest(makeRequest(uri, nil), true, false)
	assertIntsMatch(t, http.StatusBadRequest, response.Code)
	assertStringsMatch(t, errMsg, response.Body.String())
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

func doRequest(req *http.Request, gdprAllowsHostCookies bool, gdprReturnsError bool) *httptest.ResponseRecorder {
	perms := &mockPermsSetUID{
		allowHost: gdprAllowsHostCookies,
		errorHost: gdprReturnsError,
		allowPI:   true,
	}
	cfg := config.Configuration{}
	endpoint := NewSetUIDEndpoint(cfg.HostCookie, perms, analyticsConf.NewPBSAnalytics(&cfg.Analytics), metricsConf.NewMetricsEngine(&cfg, openrtb_ext.BidderList()))
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
	assertIntsMatch(t, 2, len(res))
	httpCookie := http.Cookie{
		Name:  "uids",
		Value: res[1],
	}
	return usersync.ParsePBSCookie(&httpCookie)
}

func assertIntsMatch(t *testing.T, expected int, actual int) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %d, got %d", expected, actual)
	}
}

func assertBoolsMatch(t *testing.T, expected bool, actual bool) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %t, got %t", expected, actual)
	}
}

func assertStringsMatch(t *testing.T, expected string, actual string) {
	t.Helper()
	if expected != actual {
		t.Errorf(`Expected "%s", got "%s"`, expected, actual)
	}
}

func assertSyncValue(t *testing.T, cookie *usersync.PBSCookie, family string, expectedValue string) {
	got, _, _ := cookie.GetUID(family)
	assertStringsMatch(t, expectedValue, got)
}

type mockPermsSetUID struct {
	allowHost bool
	errorHost bool
	allowPI   bool
}

func (g *mockPermsSetUID) HostCookiesAllowed(ctx context.Context, consent string) (bool, error) {
	var err error
	if g.errorHost {
		err = errors.New("something went wrong")
	}
	return g.allowHost, err
}

func (g *mockPermsSetUID) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, consent string) (bool, error) {
	return false, nil
}

func (g *mockPermsSetUID) PersonalInfoAllowed(ctx context.Context, bidder openrtb_ext.BidderName, PublisherID string, consent string) (bool, error) {
	return g.allowPI, nil
}
