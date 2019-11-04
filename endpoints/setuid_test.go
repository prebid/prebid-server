package endpoints

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PubMatic-OpenWrap/prebid-server/usersync"

	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"

	analyticsConf "github.com/PubMatic-OpenWrap/prebid-server/analytics/config"
	"github.com/PubMatic-OpenWrap/prebid-server/config"
	metricsConf "github.com/PubMatic-OpenWrap/prebid-server/pbsmetrics/config"
)

func TestNormalSet(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123", nil, false), true, false)
	assertIntsMatch(t, http.StatusOK, response.Code)
	assertHasSyncs(t, response, map[string]string{
		"pubmatic": "123",
	})
}

func TestUnset(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic", map[string]string{"pubmatic": "1234"}, false), true, false)
	assertIntsMatch(t, http.StatusOK, response.Code)
	assertHasSyncs(t, response, nil)
}

func TestMergeSet(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123", map[string]string{"rubicon": "def"}, false), true, false)
	assertIntsMatch(t, http.StatusOK, response.Code)
	assertHasSyncs(t, response, map[string]string{
		"pubmatic": "123",
		"rubicon":  "def",
	})
}

func TestGDPRPrevention(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123", nil, false), false, false)
	assertIntsMatch(t, http.StatusOK, response.Code)
	assertStringsMatch(t, "The gdpr_consent string prevents cookies from being saved", response.Body.String())
	assertNoCookie(t, response)
}

func TestGDPRConsentError(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123&gdpr_consent=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw", nil, false), false, true)
	assertIntsMatch(t, http.StatusBadRequest, response.Code)
	assertStringsMatch(t, "No global vendor list was available to interpret this consent string. If this is a new, valid version, it should become available soon.", response.Body.String())
	assertNoCookie(t, response)
}

func TestInapplicableGDPR(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123&gdpr=0", nil, false), false, false)
	assertIntsMatch(t, http.StatusOK, response.Code)
	assertHasSyncs(t, response, map[string]string{
		"pubmatic": "123",
	})
}

func TestExplicitGDPRPrevention(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123&gdpr=1&gdpr_consent=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw", nil, false), false, false)
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

func TestSecParam(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123", nil, true), true, false)
	assertIntsMatch(t, http.StatusOK, response.Code)
	uidsCookie := readUidsCookie(response.Header())
	assert.True(t, uidsCookie.Secure)
}

func TestNoSecParam(t *testing.T) {
	response := doRequest(makeRequest("/setuid?bidder=pubmatic&uid=123", nil, false), true, false)
	assertIntsMatch(t, http.StatusOK, response.Code)
	uidsCookie := readUidsCookie(response.Header())
	assert.False(t, uidsCookie.Secure)
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
	response := doRequest(makeRequest(uri, nil, false), true, false)
	assertIntsMatch(t, http.StatusBadRequest, response.Code)
	assertStringsMatch(t, errMsg, response.Body.String())
}

func makeRequest(uri string, existingSyncs map[string]string, addSecParam bool) *http.Request {
	request := httptest.NewRequest("GET", uri, nil)
	if len(existingSyncs) > 0 {
		pbsCookie := usersync.NewPBSCookie()
		for family, value := range existingSyncs {
			pbsCookie.TrySync(family, value)
		}
		addCookie(request, pbsCookie)
	}
	if addSecParam {
		q := request.URL.Query()
		q.Add("sec", "1")
		request.URL.RawQuery = q.Encode()
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

func readUidsCookie(h http.Header) *http.Cookie {
	cookieCount := len(h["Set-Cookie"])
	if cookieCount == 0 {
		return nil
	}
	//cookies := make([]*http.Cookie, 0, cookieCount)
	for _, line := range h["Set-Cookie"] {
		parts := strings.Split(strings.TrimSpace(line), ";")
		if len(parts) == 1 && parts[0] == "" {
			continue
		}
		parts[0] = strings.TrimSpace(parts[0])
		j := strings.Index(parts[0], "=")
		if j < 0 {
			continue
		}
		name, value := parts[0][:j], parts[0][j+1:]
		if name != "uids" {
			continue
		}
		//if !isCookieNameValid(name) {
		//	continue
		//}
		value, ok := parseCookieValue(value, true)
		if !ok {
			continue
		}
		c := &http.Cookie{
			Name:  name,
			Value: value,
			Raw:   line,
		}
		for i := 1; i < len(parts); i++ {
			parts[i] = strings.TrimSpace(parts[i])
			if len(parts[i]) == 0 {
				continue
			}

			attr, val := parts[i], ""
			if j := strings.Index(attr, "="); j >= 0 {
				attr, val = attr[:j], attr[j+1:]
			}
			lowerAttr := strings.ToLower(attr)
			val, ok = parseCookieValue(val, false)
			if !ok {
				c.Unparsed = append(c.Unparsed, parts[i])
				continue
			}
			switch lowerAttr {
			case "samesite":
				lowerVal := strings.ToLower(val)
				switch lowerVal {
				case "lax":
					c.SameSite = http.SameSiteLaxMode
				case "strict":
					c.SameSite = http.SameSiteStrictMode
				default:
					c.SameSite = http.SameSiteDefaultMode
				}
				continue
			case "secure":
				c.Secure = true
				continue
			case "httponly":
				c.HttpOnly = true
				continue
			case "domain":
				c.Domain = val
				continue
			case "max-age":
				secs, err := strconv.Atoi(val)
				if err != nil || secs != 0 && val[0] == '0' {
					break
				}
				if secs <= 0 {
					secs = -1
				}
				c.MaxAge = secs
				continue
			case "expires":
				c.RawExpires = val
				exptime, err := time.Parse(time.RFC1123, val)
				if err != nil {
					exptime, err = time.Parse("Mon, 02-Jan-2006 15:04:05 MST", val)
					if err != nil {
						c.Expires = time.Time{}
						break
					}
				}
				c.Expires = exptime.UTC()
				continue
			case "path":
				c.Path = val
				continue
			}
			c.Unparsed = append(c.Unparsed, parts[i])
		}
		return c
	}
	return nil
}

func parseCookieValue(raw string, allowDoubleQuote bool) (string, bool) {
	// Strip the quotes, if present.
	if allowDoubleQuote && len(raw) > 1 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	for i := 0; i < len(raw); i++ {
		if !validCookieValueByte(raw[i]) {
			return "", false
		}
	}
	return raw, true
}

func validCookieValueByte(b byte) bool {
	return 0x20 <= b && b < 0x7f && b != '"' && b != ';' && b != '\\'
}
