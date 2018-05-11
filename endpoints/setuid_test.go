package endpoints

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/prebid/prebid-server/usersync"

	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/prebid/prebid-server/pbsmetrics"

	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
)

func TestNormalSet(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123", nil))
	assertIntsMatch(t, http.StatusOK, response.Code)

	cookie := parseCookieString(t, response)
	assertIntsMatch(t, 1, cookie.LiveSyncCount())
	assertBoolsMatch(t, true, cookie.HasLiveSync("pubmatic"))
	assertSyncValue(t, cookie, "pubmatic", "123")
}

func TestUnset(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic", map[string]string{"pubmatic": "1234"}))
	assertIntsMatch(t, http.StatusOK, response.Code)

	cookie := parseCookieString(t, response)
	assertIntsMatch(t, 0, cookie.LiveSyncCount())
}

func TestMergeSet(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123", map[string]string{"rubicon": "def"}))
	assertIntsMatch(t, http.StatusOK, response.Code)

	cookie := parseCookieString(t, response)
	assertIntsMatch(t, 2, cookie.LiveSyncCount())
	assertBoolsMatch(t, true, cookie.HasLiveSync("pubmatic"))
	assertBoolsMatch(t, true, cookie.HasLiveSync("rubicon"))
	assertSyncValue(t, cookie, "pubmatic", "123")
	assertSyncValue(t, cookie, "rubicon", "def")
}

func TestNoBidder(t *testing.T) {
	response := doRequest(makeRequest("/setuid?uid=123", nil))
	assertIntsMatch(t, http.StatusBadRequest, response.Code)
}

func TestOptedOut(t *testing.T) {
	request := httptest.NewRequest("GET", "/setuid?bidder=pubmatic&uid=123", nil)
	cookie := usersync.NewPBSCookie()
	cookie.SetPreference(false)
	addCookie(request, cookie)
	response := doRequest(request)

	assertIntsMatch(t, http.StatusUnauthorized, response.Code)
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

func doRequest(req *http.Request) *httptest.ResponseRecorder {
	cfg := config.Configuration{}
	endpoint := NewSetUIDEndpoint(cfg.HostCookie, analyticsConf.NewPBSAnalytics(&cfg.Analytics), pbsmetrics.NewMetricsEngine(&cfg, openrtb_ext.BidderList()))
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
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func assertSyncValue(t *testing.T, cookie *usersync.PBSCookie, family string, expectedValue string) {
	got, _, _ := cookie.GetUID(family)
	assertStringsMatch(t, expectedValue, got)
}
