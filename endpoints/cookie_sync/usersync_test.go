package cookie_sync

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	usersyncers "github.com/prebid/prebid-server/usersync"
	metrics "github.com/rcrowley/go-metrics"
)

func TestCookieSyncNoCookies(t *testing.T) {
	endpoint := testableEndpoint()

	router := httprouter.New()
	router.POST("/cookie_sync", endpoint)

	csreq := cookieSyncRequest{
		UUID:    "abcdefg",
		Bidders: []string{"appnexus", "audienceNetwork", "random"},
	}
	csbuf := new(bytes.Buffer)
	err := json.NewEncoder(csbuf).Encode(&csreq)
	if err != nil {
		t.Fatalf("Encode csr failed: %v", err)
	}

	req, _ := http.NewRequest("POST", "/cookie_sync", csbuf)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Wrong status: %d", rr.Code)
	}

	csresp := cookieSyncResponse{}
	err = json.Unmarshal(rr.Body.Bytes(), &csresp)
	if err != nil {
		t.Fatalf("Unmarshal response failed: %v", err)
	}

	if csresp.UUID != csreq.UUID {
		t.Error("UUIDs didn't match")
	}

	if csresp.Status != "no_cookie" {
		t.Errorf("Expected status = no_cookie; got %s", csresp.Status)
	}

	if len(csresp.BidderStatus) != 2 {
		t.Errorf("Expected 2 bidder status rows; got %d", len(csresp.BidderStatus))
	}
}

func TestCookieSyncHasCookies(t *testing.T) {
	endpoint := testableEndpoint()

	router := httprouter.New()
	router.POST("/cookie_sync", endpoint)

	csreq := cookieSyncRequest{
		UUID:    "abcdefg",
		Bidders: []string{"appnexus", "audienceNetwork", "random"},
	}
	csbuf := new(bytes.Buffer)
	err := json.NewEncoder(csbuf).Encode(&csreq)
	if err != nil {
		t.Fatalf("Encode csr failed: %v", err)
	}

	req, _ := http.NewRequest("POST", "/cookie_sync", csbuf)

	pcs := pbs.ParsePBSCookieFromRequest(req, &config.Cookie{})
	pcs.TrySync("adnxs", "1234")
	pcs.TrySync("audienceNetwork", "2345")
	req.AddCookie(pcs.ToHTTPCookie())

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Wrong status: %d", rr.Code)
	}

	csresp := cookieSyncResponse{}
	err = json.Unmarshal(rr.Body.Bytes(), &csresp)
	if err != nil {
		t.Fatalf("Unmarshal response failed: %v", err)
	}

	if csresp.UUID != csreq.UUID {
		t.Error("UUIDs didn't match")
	}

	if csresp.Status != "ok" {
		t.Errorf("Expected status = ok; got %s", csresp.Status)
	}

	if len(csresp.BidderStatus) != 0 {
		t.Errorf("Expected 0 bidder status rows; got %d", len(csresp.BidderStatus))
	}
}

func testableEndpoint() httprouter.Handle {
	knownSyncers := map[openrtb_ext.BidderName]usersyncers.Usersyncer{
		openrtb_ext.BidderAppnexus: usersyncers.NewAppnexusSyncer("someurl.com"),
		openrtb_ext.BidderFacebook: usersyncers.NewFacebookSyncer("facebookurl.com"),
	}
	return NewEndpoint(knownSyncers, &config.Cookie{}, metrics.NewMeter())
}
