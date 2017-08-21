package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbs"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCookieSyncNoCookies(t *testing.T) {
	cfg, err := config.New()
	if err != nil {
		t.Fatalf("Unable to config: %v", err)
	}
	setupExchanges(cfg)
	router := httprouter.New()
	router.POST("/cookie_sync", cookieSync)

	csreq := cookieSyncRequest{
		UUID:    "abcdefg",
		Bidders: []string{"appnexus", "audienceNetwork", "random"},
	}
	csbuf := new(bytes.Buffer)
	err = json.NewEncoder(csbuf).Encode(&csreq)
	if err != nil {
		t.Fatalf("Encode csr failed: %v", err)
	}

	req, _ :=  http.NewRequest("POST", "/cookie_sync", csbuf)
	rr :=httptest.NewRecorder()
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
		t.Errorf("UUIDs didn't match")
	}

	if csresp.Status != "no_cookie" {
		t.Errorf("Expected status = no_cookie; got %s", csresp.Status)
	}

	if len(csresp.BidderStatus) != 2 {
		t.Errorf("Expected 2 bidder status rows; got %d", len(csresp.BidderStatus))
	}
}

func TestCookieSyncHasCookies(t *testing.T) {
	cfg, err := config.New()
	if err != nil {
		t.Fatalf("Unable to config: %v", err)
	}
	setupExchanges(cfg)
	router := httprouter.New()
	router.POST("/cookie_sync", cookieSync)

	csreq := cookieSyncRequest{
		UUID:    "abcdefg",
		Bidders: []string{"appnexus", "audienceNetwork", "random"},
	}
	csbuf := new(bytes.Buffer)
	err = json.NewEncoder(csbuf).Encode(&csreq)
	if err != nil {
		t.Fatalf("Encode csr failed: %v", err)
	}

	req, _ := http.NewRequest("POST", "/cookie_sync", csbuf)

	pcs := pbs.ParseUIDCookie(req)
	pcs.UIDs["adnxs"] = "1234"
	pcs.UIDs["audienceNetwork"] = "2345"
	j, _ := json.Marshal(pcs)
	b64 := base64.URLEncoding.EncodeToString(j)
	uid_cookie := http.Cookie{Name: "uids", Value: b64}
	req.AddCookie(&uid_cookie)

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
		t.Errorf("UUIDs didn't match")
	}

	if csresp.Status != "ok" {
		t.Errorf("Expected status = ok; got %s", csresp.Status)
	}

	if len(csresp.BidderStatus) != 0 {
		t.Errorf("Expected 0 bidder status rows; got %d", len(csresp.BidderStatus))
	}
}
