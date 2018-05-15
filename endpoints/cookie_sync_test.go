package endpoints

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/buger/jsonparser"

	"github.com/julienschmidt/httprouter"
	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/usersync"
)

func TestCookieSyncNoCookies(t *testing.T) {
	rr := doPost(`{"bidders":["appnexus", "audienceNetwork", "random"]}`, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("Wrong status: %d (%s)", rr.Code, rr.Body)
	}
	assertSyncsExist(t, rr.Body.Bytes(), "appnexus", "audienceNetwork")
	assertStatus(t, rr.Body.Bytes(), "no_cookie")
}

func TestCookieSyncHasCookies(t *testing.T) {
	rr := doPost(`{"bidders":["appnexus", "audienceNetwork", "random"]}`, map[string]string{
		"adnxs":           "1234",
		"audienceNetwork": "2345",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("Wrong status: %d", rr.Code)
	}
	assertSyncsExist(t, rr.Body.Bytes())
	assertStatus(t, rr.Body.Bytes(), "ok")
}

// Make sure that an empty bidders array returns no syncs
func TestCookieSyncEmptyBidders(t *testing.T) {
	rr := doPost(`{"bidders": []}`, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("Wrong status: %d (%s)", rr.Code, rr.Body)
	}
	assertSyncsExist(t, rr.Body.Bytes())
	assertStatus(t, rr.Body.Bytes(), "no_cookie")
}

// Make sure that all syncs are returned if "bidders" isn't a key
func TestCookieSyncNoBidders(t *testing.T) {
	rr := doPost("{}", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("Wrong status: %d (%s)", rr.Code, rr.Body)
	}
	assertSyncsExist(t, rr.Body.Bytes(), "appnexus", "audienceNetwork", "lifestreet", "pubmatic")
	assertStatus(t, rr.Body.Bytes(), "no_cookie")
}

func doPost(body string, existingSyncs map[string]string) *httptest.ResponseRecorder {
	endpoint := testableEndpoint()
	router := httprouter.New()
	router.POST("/cookie_sync", endpoint)
	req, _ := http.NewRequest("POST", "/cookie_sync", strings.NewReader(body))
	if len(existingSyncs) > 0 {
		pcs := pbs.NewPBSCookie()
		for bidder, uid := range existingSyncs {
			pcs.TrySync(bidder, uid)
		}
		req.AddCookie(pcs.ToHTTPCookie(90 * 24 * time.Hour))
	}

	rr := httptest.NewRecorder()
	endpoint(rr, req, nil)
	return rr
}

func testableEndpoint() httprouter.Handle {
	knownSyncers := map[openrtb_ext.BidderName]usersync.Usersyncer{
		openrtb_ext.BidderAppnexus:   usersync.NewAppnexusSyncer("someurl.com"),
		openrtb_ext.BidderFacebook:   usersync.NewFacebookSyncer("facebookurl.com"),
		openrtb_ext.BidderLifestreet: usersync.NewLifestreetSyncer("anotherurl.com"),
		openrtb_ext.BidderPubmatic:   usersync.NewPubmaticSyncer("thaturl.com"),
	}
	return NewCookieSyncEndpoint(knownSyncers, &config.Cookie{}, &pbsmetrics.DummyMetricsEngine{}, analyticsConf.NewPBSAnalytics(&config.Analytics{}))
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
