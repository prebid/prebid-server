package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mxmCherry/openrtb"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbs"
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

func TestBidSizeValidate(t *testing.T) {

	bids := make(pbs.PBSBidSlice, 0)

	//bid_1 will be rejected due to undefined size when adunit has multiple sizes
	bid_1 := pbs.PBSBid{
		BidID:      "test_bidid1",
		AdUnitCode: "test_adunitcode",
		BidderCode: "randNetwork",
		Price:      1.05,
		Adm:        "test_adm",
		//Width:             100,
		//Height:            100,
		CreativeMediaType: "banner",
	}

	bids = append(bids, &bid_1)

	//bid_2 will be considered a normal ideal banner bid
	bid_2 := pbs.PBSBid{
		BidID:             "test_bidid2",
		AdUnitCode:        "test_adunitcode2",
		BidderCode:        "randNetwork",
		Price:             1.05,
		Adm:               "test_adm",
		Width:             100,
		Height:            100,
		CreativeMediaType: "banner",
	}

	bids = append(bids, &bid_2)

	//bid_3 will have it's dimensions set based on sizes defined in request
	bid_3 := pbs.PBSBid{
		BidID:      "test_bidid3",
		AdUnitCode: "test_adunitcode3",
		BidderCode: "randNetwork",
		Price:      1.05,
		Adm:        "test_adm",
		//Width:             200,
		//Height:            200,
		CreativeMediaType: "banner",
	}

	bids = append(bids, &bid_3)

	//bid_4 will be ignored as it's a video creative type
	bid_4 := pbs.PBSBid{
		BidID:      "test_bidid_video",
		AdUnitCode: "test_adunitcode2",
		BidderCode: "randNetwork",
		Price:      1.05,
		Adm:        "test_adm",
		//Width:             400,
		//Height:            400,
		CreativeMediaType: "video",
	}

	bids = append(bids, &bid_4)

	mybidder := pbs.PBSBidder{
		BidderCode: "randNetwork",
		AdUnitCode: "test_adunitcode",
		AdUnits: []pbs.PBSAdUnit{
			pbs.PBSAdUnit{
				BidID: "test_bidid1",
				Sizes: []openrtb.Format{
					openrtb.Format{
						W: 350,
						H: 250,
					},
					openrtb.Format{
						W: 300,
						H: 50,
					},
				},
				Code: "sample_test_code",
				MediaTypes: []pbs.MediaType{
					pbs.MEDIA_TYPE_BANNER,
				},
			},
			pbs.PBSAdUnit{
				BidID: "test_bidid2",
				Sizes: []openrtb.Format{
					openrtb.Format{
						W: 100,
						H: 100,
					},
				},
				Code: "sample_test_code",
				MediaTypes: []pbs.MediaType{
					pbs.MEDIA_TYPE_BANNER,
				},
			},
			pbs.PBSAdUnit{
				BidID: "test_bidid3",
				Sizes: []openrtb.Format{
					openrtb.Format{
						W: 200,
						H: 200,
					},
				},
				Code: "sample_test_code",
				MediaTypes: []pbs.MediaType{
					pbs.MEDIA_TYPE_BANNER,
				},
			},
			pbs.PBSAdUnit{
				BidID: "test_bidid_video",
				Sizes: []openrtb.Format{
					openrtb.Format{
						W: 400,
						H: 400,
					},
				},
				Code: "sample_test_code",
				MediaTypes: []pbs.MediaType{
					pbs.MEDIA_TYPE_VIDEO,
				},
			},
		},
	}

	bids = checkForValidBidSize(bids, &mybidder)

	testdata, _ := json.MarshalIndent(bids, "", "   ")
	if len(bids) != 3 {
		t.Errorf("Detected returned bid list did not contain only 3 bid objects as expected.\nBelow is the contents of the bid list\n%v", string(testdata))
	}

	for _, bid := range bids {
		if bid.BidID == "test_bidid3" {
			if bid.Width == 0 && bid.Height == 0 {
				t.Errorf("Detected the Width & Height attributes in test bidID %v were not set to the dimensions used from the mybidder object", bid.BidID)
			}
		}
	}
}
