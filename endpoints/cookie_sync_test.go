package endpoints

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/buger/jsonparser"

	"github.com/julienschmidt/httprouter"
	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/openrtb_ext"
	metricsConf "github.com/prebid/prebid-server/pbsmetrics/config"
	"github.com/prebid/prebid-server/usersync"
	"github.com/prebid/prebid-server/usersync/usersyncers"
)

func TestCookieSyncNoCookies(t *testing.T) {
	rr := doPost(`{"bidders":["appnexus", "audienceNetwork", "random"]}`, nil, true, syncersForTest())
	assertIntsMatch(t, http.StatusOK, rr.Code)
	assertSyncsExist(t, rr.Body.Bytes(), "appnexus", "audienceNetwork")
	assertStatus(t, rr.Body.Bytes(), "no_cookie")
}

func TestGDPRPreventsCookie(t *testing.T) {
	rr := doPost(`{"bidders":["appnexus", "pubmatic"]}`, nil, false, syncersForTest())
	assertIntsMatch(t, http.StatusOK, rr.Code)

	assertSyncsExist(t, rr.Body.Bytes())
	assertStatus(t, rr.Body.Bytes(), "no_cookie")
}

func TestGDPRPreventsBidders(t *testing.T) {
	rr := doPost(`{"gdpr":1,"bidders":["appnexus", "pubmatic", "lifestreet"],"gdpr_consent":"BOONs2HOONs2HABABBENAGgAAAAPrABACGA"}`, nil, true, map[openrtb_ext.BidderName]usersync.Usersyncer{
		openrtb_ext.BidderLifestreet: usersyncers.NewLifestreetSyncer("someurl.com"),
	})
	assertIntsMatch(t, http.StatusOK, rr.Code)
	assertSyncsExist(t, rr.Body.Bytes(), "lifestreet")
	assertStatus(t, rr.Body.Bytes(), "no_cookie")
}

func TestGDPRIgnoredIfZero(t *testing.T) {
	rr := doPost(`{"gdpr":0,"bidders":["appnexus", "pubmatic"]}`, nil, false, nil)
	assertIntsMatch(t, http.StatusOK, rr.Code)

	assertSyncsExist(t, rr.Body.Bytes(), "appnexus", "pubmatic")
	assertStatus(t, rr.Body.Bytes(), "no_cookie")
}

func TestGDPRConsentRequired(t *testing.T) {
	rr := doPost(`{"gdpr":1,"bidders":["appnexus", "pubmatic"]}`, nil, false, nil)
	assertIntsMatch(t, http.StatusBadRequest, rr.Code)
	assertStringsMatch(t, "gdpr_consent is required if gdpr=1\n", rr.Body.String())
}

func TestCookieSyncHasCookies(t *testing.T) {
	rr := doPost(`{"bidders":["appnexus", "audienceNetwork", "random"]}`, map[string]string{
		"adnxs":           "1234",
		"audienceNetwork": "2345",
	}, true, syncersForTest())
	assertIntsMatch(t, http.StatusOK, rr.Code)
	assertSyncsExist(t, rr.Body.Bytes())
	assertStatus(t, rr.Body.Bytes(), "ok")
}

// Make sure that an empty bidders array returns no syncs
func TestCookieSyncEmptyBidders(t *testing.T) {
	rr := doPost(`{"bidders": []}`, nil, true, syncersForTest())
	assertIntsMatch(t, http.StatusOK, rr.Code)
	assertSyncsExist(t, rr.Body.Bytes())
	assertStatus(t, rr.Body.Bytes(), "no_cookie")
}

// Make sure that all syncs are returned if "bidders" isn't a key
func TestCookieSyncNoBidders(t *testing.T) {
	rr := doPost("{}", nil, true, syncersForTest())
	assertIntsMatch(t, http.StatusOK, rr.Code)
	assertSyncsExist(t, rr.Body.Bytes(), "appnexus", "audienceNetwork", "lifestreet", "pubmatic")
	assertStatus(t, rr.Body.Bytes(), "no_cookie")
}

func TestCookieSyncNoCookiesBrokenGDPR(t *testing.T) {
	rr := doConfigurablePost(`{"bidders":["appnexus", "audienceNetwork", "random"],"gdpr_consent":"GLKHGKGKKGK"}`, nil, true, map[openrtb_ext.BidderName]usersync.Usersyncer{}, config.GDPR{UsersyncIfAmbiguous: true})
	assertIntsMatch(t, http.StatusOK, rr.Code)
	assertSyncsExist(t, rr.Body.Bytes(), "appnexus", "audienceNetwork")
	assertStatus(t, rr.Body.Bytes(), "no_cookie")
}

func doPost(body string, existingSyncs map[string]string, gdprHostConsent bool, gdprBidders map[openrtb_ext.BidderName]usersync.Usersyncer) *httptest.ResponseRecorder {
	return doConfigurablePost(body, existingSyncs, gdprHostConsent, gdprBidders, config.GDPR{})
}

func doConfigurablePost(body string, existingSyncs map[string]string, gdprHostConsent bool, gdprBidders map[openrtb_ext.BidderName]usersync.Usersyncer, cfgGDPR config.GDPR) *httptest.ResponseRecorder {
	endpoint := testableEndpoint(mockPermissions(gdprHostConsent, gdprBidders), cfgGDPR)
	router := httprouter.New()
	router.POST("/cookie_sync", endpoint)
	req, _ := http.NewRequest("POST", "/cookie_sync", strings.NewReader(body))
	if len(existingSyncs) > 0 {
		pcs := usersync.NewPBSCookie()
		for bidder, uid := range existingSyncs {
			pcs.TrySync(bidder, uid)
		}
		req.AddCookie(pcs.ToHTTPCookie(90 * 24 * time.Hour))
	}

	rr := httptest.NewRecorder()
	endpoint(rr, req, nil)
	return rr
}

func testableEndpoint(perms gdpr.Permissions, cfgGDPR config.GDPR) httprouter.Handle {
	return NewCookieSyncEndpoint(syncersForTest(), &config.Configuration{GDPR: cfgGDPR}, perms, &metricsConf.DummyMetricsEngine{}, analyticsConf.NewPBSAnalytics(&config.Analytics{}))
}

func syncersForTest() map[openrtb_ext.BidderName]usersync.Usersyncer {
	return map[openrtb_ext.BidderName]usersync.Usersyncer{
		openrtb_ext.BidderAppnexus:   usersyncers.NewAppnexusSyncer("someurl.com"),
		openrtb_ext.BidderFacebook:   usersyncers.NewFacebookSyncer("facebookurl.com"),
		openrtb_ext.BidderLifestreet: usersyncers.NewLifestreetSyncer("anotherurl.com"),
		openrtb_ext.BidderPubmatic:   usersyncers.NewPubmaticSyncer("thaturl.com"),
	}
}

func assertSyncsExist(t *testing.T, responseBody []byte, expectedBidders ...string) {
	t.Helper()
	assertSameElements(t, expectedBidders, parseSyncs(t, responseBody))
}

func assertStatus(t *testing.T, responseBody []byte, expected string) {
	t.Helper()
	val, err := jsonparser.GetString(responseBody, "status")
	if err != nil {
		t.Errorf("response.status was not a string. Error was %v", err)
		return
	}
	if val != expected {
		t.Errorf("response.status was %s, but expected %s", val, expected)
	}
}

func parseSyncs(t *testing.T, response []byte) []string {
	t.Helper()
	var syncs []string
	jsonparser.ArrayEach(response, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if dataType != jsonparser.Object {
			t.Errorf("response.bidder_status contained unexpected element of type %v.", dataType)
		}
		if val, err := jsonparser.GetString(value, "bidder"); err != nil {
			t.Errorf("response.bidder_status[?].bidder was not a string. Value was %s", string(value))
		} else {
			syncs = append(syncs, val)
		}
	}, "bidder_status")
	return syncs
}

func assertSameElements(t *testing.T, expected []string, actual []string) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Errorf("expected %v, but got %v.", expected, actual)
		return
	}
	for _, expectedVal := range expected {
		seen := false
		for _, actualVal := range actual {
			if expectedVal == actualVal {
				seen = true
				break
			}
		}
		if !seen {
			t.Errorf("Expected sync from %s, but it wasn't in the response.", expectedVal)
		}
	}
}

func mockPermissions(allowHost bool, allowedBidders map[openrtb_ext.BidderName]usersync.Usersyncer) gdpr.Permissions {
	return &gdprPerms{
		allowHost:      allowHost,
		allowedBidders: allowedBidders,
	}
}

type gdprPerms struct {
	allowHost      bool
	allowedBidders map[openrtb_ext.BidderName]usersync.Usersyncer
}

func (g *gdprPerms) HostCookiesAllowed(ctx context.Context, consent string) (bool, error) {
	return g.allowHost, nil
}

func (g *gdprPerms) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, consent string) (bool, error) {
	_, ok := g.allowedBidders[bidder]
	return ok, nil
}

func (g *gdprPerms) PersonalInfoAllowed(ctx context.Context, bidder openrtb_ext.BidderName, consent string) (bool, error) {
	return true, nil
}
