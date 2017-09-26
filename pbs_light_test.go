package main

import (
	"bytes"
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/cache/dummycache"
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

	pcs := pbs.ParsePBSCookieFromRequest(req)
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
		t.Errorf("UUIDs didn't match")
	}

	if csresp.Status != "ok" {
		t.Errorf("Expected status = ok; got %s", csresp.Status)
	}

	if len(csresp.BidderStatus) != 0 {
		t.Errorf("Expected 0 bidder status rows; got %d", len(csresp.BidderStatus))
	}
}

func TestSortBidsAndAddKeywordsForMobile(t *testing.T) {
	body := []byte(`{
	   "max_key_length":20,
	   "user":{
	      "gender":"F",
	      "buyeruid":"test_buyeruid",
	      "yob":2000,
	      "id":"testid"
	   },
	   "prebid_version":"0.21.0-pre",
	   "sort_bids":1,
	   "ad_units":[
	      {
	         "sizes":[
	            {
	               "w":300,
	               "h":250
	            }
	         ],
	         "config_id":"ad5ffb41-3492-40f3-9c25-ade093eb4e5f",
	         "code":"test_adunitcode"
	      }
	   ],
	   "cache_markup":1,
	   "app":{
	      "bundle":"AppNexus.PrebidMobileDemo",
	      "ver":"0.0.1"
	   },
	   "sdk":{
	      "version":"0.0.1",
	      "platform":"iOS",
	      "source":"prebid-mobile"
	   },
	   "device":{
	      "ifa":"test_device_ifa",
	      "osv":"9.3.5",
	      "os":"iOS",
	      "make":"Apple",
	      "model":"iPhone6,1"
	   },
	   "tid":"abcd",
	   "account_id":"aecd6ef7-b992-4e99-9bb8-65e2d984e1dd"
	}
    `)
	r := httptest.NewRequest("POST", "/auction", bytes.NewBuffer(body))
	d, _ := dummycache.New()

	pbs_req, err := pbs.ParsePBSRequest(r, d)
	if err != nil {
		t.Errorf("Unexpected error on parsing %v", err)
	}

	bids := make(pbs.PBSBidSlice, 0)

	fb_bid := pbs.PBSBid{
		BidID:      "test_bidid",
		AdUnitCode: "test_adunitcode",
		BidderCode: "audienceNetwork",
		Price:      1.05,
		Adm:        "test_adm",
		Width:      300,
		Height:     250,
	}
	bids = append(bids, &fb_bid)
	an_bid := pbs.PBSBid{
		BidID:      "test_bidid2",
		AdUnitCode: "test_adunitcode",
		BidderCode: "appnexus",
		Price:      1.00,
		Adm:        "test_adm",
		Width:      300,
		Height:     250,
	}
	bids = append(bids, &an_bid)
	pbs_resp := pbs.PBSResponse{
		Bids: bids,
	}
	sortBidsAddKeywordsMobile(pbs_resp.Bids, pbs_req, "")

	for _, bid := range bids {
		if bid.AdServerTargeting == nil {
			t.Errorf("Ad server targeting should not be nil")
		}
		if bid.BidderCode == "audienceNetwork" {
			if bid.AdServerTargeting["hb_creative_loadtype"] != "demand_sdk" {
				t.Errorf("Facebook bid should have demand_sdk as hb_creative_loadtype in ad server targeting")
			}
		}
	}
}
