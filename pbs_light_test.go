package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/mxmCherry/openrtb"
	"github.com/rcrowley/go-metrics"

	"context"
	"io/ioutil"
	"os"
	"time"

	"fmt"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/prebid_cache_client"
	usersyncers "github.com/prebid/prebid-server/usersync"
	"github.com/spf13/viper"
)

const adapterDirectory = "adapters"

func TestCookieSyncNoCookies(t *testing.T) {
	endpoint := testableEndpoint()

	router := httprouter.New()
	router.POST("/cookie_sync", endpoint)

	csreq := cookieSyncRequest{
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
		t.Fatalf("Wrong status: %d (%s)", rr.Code, rr.Body)
	}

	csresp := cookieSyncResponse{}
	err = json.Unmarshal(rr.Body.Bytes(), &csresp)
	if err != nil {
		t.Fatalf("Unmarshal response failed: %v", err)
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
	req.AddCookie(pcs.ToHTTPCookie(90 * 24 * time.Hour))

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

	if csresp.Status != "ok" {
		t.Errorf("Expected status = ok; got %s", csresp.Status)
	}

	if len(csresp.BidderStatus) != 0 {
		t.Errorf("Expected 0 bidder status rows; got %d", len(csresp.BidderStatus))
	}
}

func TestCookieSyncEmptyBidders(t *testing.T) {
	endpoint := testableEndpoint()

	router := httprouter.New()
	router.POST("/cookie_sync", endpoint)

	// First test a declared empty bidders returns no syncs
	csreq := []byte("{\"bidders\": []}")
	csbuf := bytes.NewBuffer(csreq)

	req, _ := http.NewRequest("POST", "/cookie_sync", csbuf)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Wrong status: %d (%s)", rr.Code, rr.Body)
	}

	csresp := cookieSyncResponse{}
	err := json.Unmarshal(rr.Body.Bytes(), &csresp)
	if err != nil {
		t.Fatalf("Unmarshal response failed: %v", err)
	}

	if csresp.Status != "no_cookie" {
		t.Errorf("Expected status = no_cookie; got %s", csresp.Status)
	}

	if len(csresp.BidderStatus) != 0 {
		t.Errorf("Expected 0 bidder status rows; got %d", len(csresp.BidderStatus))
	}
}

func TestCookieSyncNoBidders(t *testing.T) {
	endpoint := testableEndpoint()

	router := httprouter.New()
	router.POST("/cookie_sync", endpoint)

	// Now test a missing bidders returns all syncs
	csreq := []byte("{}")
	csbuf := bytes.NewBuffer(csreq)

	req, _ := http.NewRequest("POST", "/cookie_sync", csbuf)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Wrong status: %d (%s)", rr.Code, rr.Body)
	}

	csresp := cookieSyncResponse{}
	err := json.Unmarshal(rr.Body.Bytes(), &csresp)
	if err != nil {
		t.Fatalf("Unmarshal response failed: %v", err)
	}

	if csresp.Status != "no_cookie" {
		t.Errorf("Expected status = no_cookie; got %s", csresp.Status)
	}

	if len(csresp.BidderStatus) != 4 {
		t.Errorf("Expected %d bidder status rows; got %d", 4, len(csresp.BidderStatus))
	}

}

func testableEndpoint() httprouter.Handle {
	knownSyncers := map[openrtb_ext.BidderName]usersyncers.Usersyncer{
		openrtb_ext.BidderAppnexus:   usersyncers.NewAppnexusSyncer("someurl.com"),
		openrtb_ext.BidderFacebook:   usersyncers.NewFacebookSyncer("facebookurl.com"),
		openrtb_ext.BidderLifestreet: usersyncers.NewLifestreetSyncer("anotherurl.com"),
		openrtb_ext.BidderPubmatic:   usersyncers.NewPubmaticSyncer("thaturl.com"),
	}
	return (&cookieSyncDeps{knownSyncers, &config.Cookie{}, metrics.NewMeter()}).CookieSync
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
	hcs := pbs.HostCookieSettings{}

	pbs_req, err := pbs.ParsePBSRequest(r, d, &hcs)
	if err != nil {
		t.Errorf("Unexpected error on parsing %v", err)
	}

	bids := make(pbs.PBSBidSlice, 0)

	fb_bid := pbs.PBSBid{
		BidID:      "test_bidid",
		AdUnitCode: "test_adunitcode",
		BidderCode: "audienceNetwork",
		Price:      2.00,
		Adm:        "test_adm",
		Width:      300,
		Height:     250,
		CacheID:    "test_cache_id1",
		DealId:     "2345",
	}
	bids = append(bids, &fb_bid)
	an_bid := pbs.PBSBid{
		BidID:      "test_bidid2",
		AdUnitCode: "test_adunitcode",
		BidderCode: "appnexus",
		Price:      1.00,
		Adm:        "test_adm",
		Width:      320,
		Height:     50,
		CacheID:    "test_cache_id2",
		DealId:     "1234",
	}
	bids = append(bids, &an_bid)
	rb_bid := pbs.PBSBid{
		BidID:      "test_bidid2",
		AdUnitCode: "test_adunitcode",
		BidderCode: "rubicon",
		Price:      1.00,
		Adm:        "test_adm",
		Width:      300,
		Height:     250,
		CacheID:    "test_cache_id2",
		DealId:     "7890",
	}
	rb_bid.AdServerTargeting = map[string]string{
		"rpfl_1001": "15_tier0100",
	}
	bids = append(bids, &rb_bid)
	nosize_bid := pbs.PBSBid{
		BidID:      "test_bidid2",
		AdUnitCode: "test_adunitcode",
		BidderCode: "nosizebidder",
		Price:      1.00,
		Adm:        "test_adm",
		CacheID:    "test_cache_id2",
	}
	bids = append(bids, &nosize_bid)
	nodeal_bid := pbs.PBSBid{
		BidID:      "test_bidid2",
		AdUnitCode: "test_adunitcode",
		BidderCode: "nodeal",
		Price:      1.00,
		Adm:        "test_adm",
		CacheID:    "test_cache_id2",
	}
	bids = append(bids, &nodeal_bid)
	pbs_resp := pbs.PBSResponse{
		Bids: bids,
	}
	sortBidsAddKeywordsMobile(pbs_resp.Bids, pbs_req, "")

	for _, bid := range bids {
		if bid.AdServerTargeting == nil {
			t.Error("Ad server targeting should not be nil")
		}
		if bid.BidderCode == "audienceNetwork" {
			if bid.AdServerTargeting["hb_creative_loadtype"] != "demand_sdk" {
				t.Error("Facebook bid should have demand_sdk as hb_creative_loadtype in ad server targeting")
			}
			if bid.AdServerTargeting["hb_size"] != "300x250" {
				t.Error("hb_size key was not parsed correctly")
			}
			if bid.AdServerTargeting["hb_pb"] != "2.00" {
				t.Error("hb_pb key was not parsed correctly ", bid.AdServerTargeting["hb_pb"])
			}

			if bid.AdServerTargeting["hb_cache_id"] != "test_cache_id1" {
				t.Error("hb_cache_id key was not parsed correctly")
			}
			if bid.AdServerTargeting["hb_bidder"] != "audienceNetwork" {
				t.Error("hb_bidder key was not parsed correctly")
			}
			if bid.AdServerTargeting["hb_deal"] != "2345" {
				t.Error("hb_deal_id key was not parsed correctly ")
			}
		}
		if bid.BidderCode == "appnexus" {
			if bid.AdServerTargeting["hb_size_appnexus"] != "320x50" {
				t.Error("hb_size key for appnexus bidder was not parsed correctly")
			}
			if bid.AdServerTargeting["hb_cache_id_appnexus"] != "test_cache_id2" {
				t.Error("hb_cache_id key for appnexus bidder was not parsed correctly")
			}
			if bid.AdServerTargeting["hb_bidder_appnexus"] != "appnexus" {
				t.Error("hb_bidder key for appnexus bidder was not parsed correctly")
			}
			if bid.AdServerTargeting["hb_pb_appnexus"] != "1.00" {
				t.Error("hb_pb key for appnexus bidder was not parsed correctly")
			}
			if bid.AdServerTargeting["hb_pb"] != "" {
				t.Error("hb_pb key was parsed for two bidders")
			}
			if bid.AdServerTargeting["hb_deal_appnexus"] != "1234" {
				t.Errorf("hb_deal_id_appnexus was not parsed correctly %v", bid.AdServerTargeting["hb_deal_id_appnexus"])
			}
		}
		if bid.BidderCode == "rubicon" {
			if bid.AdServerTargeting["rpfl_1001"] != "15_tier0100" {
				t.Error("custom ad_server_targeting KVPs from adapter were not preserved")
			}
		}
		if bid.BidderCode == "nosizebidder" {
			if _, exists := bid.AdServerTargeting["hb_size_nosizebidder"]; exists {
				t.Error("hb_size key for nosize bidder was not parsed correctly", bid.AdServerTargeting)
			}
		}
		if bid.BidderCode == "nodeal" {
			if _, exists := bid.AdServerTargeting["hb_deal_nodeal"]; exists {
				t.Error("hb_deal_id key for nodeal bidder was not parsed correctly")
			}
		}
	}
}

var (
	MaxValueLength = 1024 * 10
	MaxNumValues   = 10
)

type responseObject struct {
	UUID string `json:"uuid"`
}

type response struct {
	Responses []responseObject `json:"responses"`
}

type putAnyObject struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

type putAnyRequest struct {
	Puts []putAnyObject `json:"puts"`
}

func DummyPrebidCacheServer(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read the request body.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	var put putAnyRequest

	err = json.Unmarshal(body, &put)
	if err != nil {
		http.Error(w, "Request body "+string(body)+" is not valid JSON.", http.StatusBadRequest)
		return
	}

	if len(put.Puts) > MaxNumValues {
		http.Error(w, fmt.Sprintf("More keys than allowed: %d", MaxNumValues), http.StatusBadRequest)
		return
	}

	resp := response{
		Responses: make([]responseObject, len(put.Puts)),
	}
	for i, p := range put.Puts {
		resp.Responses[i].UUID = fmt.Sprintf("UUID-%d", i+1) // deterministic for testing
		if len(p.Value) > MaxValueLength {
			http.Error(w, fmt.Sprintf("Value is larger than allowed size: %d", MaxValueLength), http.StatusBadRequest)
			return
		}
		if len(p.Value) == 0 {
			http.Error(w, "Missing value.", http.StatusBadRequest)
			return
		}
		if p.Type != "xml" && p.Type != "json" {
			http.Error(w, fmt.Sprintf("Type must be one of [\"json\", \"xml\"]. Found %v", p.Type), http.StatusBadRequest)
			return
		}
	}

	bytes, err := json.Marshal(&resp)
	if err != nil {
		http.Error(w, "Failed to serialize UUIDs into JSON.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

func TestCacheVideoOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyPrebidCacheServer))
	defer server.Close()

	bids := make(pbs.PBSBidSlice, 0)
	fbBid := pbs.PBSBid{
		BidID:             "test_bidid0",
		AdUnitCode:        "test_adunitcode0",
		BidderCode:        "audienceNetwork",
		Price:             2.00,
		Adm:               "fb_test_adm",
		Width:             300,
		Height:            250,
		DealId:            "2345",
		CreativeMediaType: "video",
	}
	bids = append(bids, &fbBid)
	anBid := pbs.PBSBid{
		BidID:             "test_bidid1",
		AdUnitCode:        "test_adunitcode1",
		BidderCode:        "appnexus",
		Price:             1.00,
		Adm:               "an_test_adm",
		Width:             320,
		Height:            50,
		DealId:            "1234",
		CreativeMediaType: "banner",
	}
	bids = append(bids, &anBid)
	rbBannerBid := pbs.PBSBid{
		BidID:             "test_bidid2",
		AdUnitCode:        "test_adunitcode2",
		BidderCode:        "rubicon",
		Price:             1.00,
		Adm:               "rb_banner_test_adm",
		Width:             300,
		Height:            250,
		DealId:            "7890",
		CreativeMediaType: "banner",
	}
	bids = append(bids, &rbBannerBid)
	rbVideoBid1 := pbs.PBSBid{
		BidID:             "test_bidid3",
		AdUnitCode:        "test_adunitcode3",
		BidderCode:        "rubicon",
		Price:             1.00,
		Adm:               "rb_video_test_adm1",
		Width:             300,
		Height:            250,
		DealId:            "7890",
		CreativeMediaType: "video",
	}
	bids = append(bids, &rbVideoBid1)
	rbVideoBid2 := pbs.PBSBid{
		BidID:             "test_bidid4",
		AdUnitCode:        "test_adunitcode4",
		BidderCode:        "rubicon",
		Price:             1.00,
		Adm:               "rb_video_test_adm2",
		Width:             300,
		Height:            250,
		DealId:            "7890",
		CreativeMediaType: "video",
	}
	bids = append(bids, &rbVideoBid2)

	ctx := context.TODO()
	w := httptest.NewRecorder()
	cfg, err := config.New(viper.New())
	if err != nil {
		t.Fatal(err.Error())
	}
	syncers := usersyncers.NewSyncerMap(cfg)
	prebid_cache_client.InitPrebidCache(server.URL)
	cacheVideoOnly(bids, ctx, w, &auctionDeps{cfg, syncers})
	if bids[0].CacheID != "UUID-1" {
		t.Errorf("UUID was '%s', should have been 'UUID-1'", bids[0].CacheID)
	}
	if bids[1].CacheID != "" {
		t.Errorf("UUID was '%s', should have been empty", bids[1].CacheID)
	}
	if bids[2].CacheID != "" {
		t.Errorf("UUID was '%s', should have been empty", bids[2].CacheID)
	}
	if bids[3].CacheID != "UUID-2" {
		t.Errorf("First object UUID was '%s', should have been 'UUID-2'", bids[3].CacheID)
	}
	if bids[4].CacheID != "UUID-3" {
		t.Errorf("Second object UUID was '%s', should have been 'UUID-3'", bids[4].CacheID)
	}
}

func TestBidSizeValidate(t *testing.T) {

	bids := make(pbs.PBSBidSlice, 0)

	//bid_1 will be rejected due to undefined size when adunit has multiple sizes
	bid_1 := pbs.PBSBid{
		BidID:      "test_bidid1",
		AdUnitCode: "test_adunitcode1",
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
		AdUnitCode: "test_adunitcode_video",
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
			{
				BidID: "test_bidid1",
				Sizes: []openrtb.Format{
					{
						W: 350,
						H: 250,
					},
					{
						W: 300,
						H: 50,
					},
				},
				Code: "test_adunitcode1",
				MediaTypes: []pbs.MediaType{
					pbs.MEDIA_TYPE_BANNER,
				},
			},
			{
				BidID: "test_bidid2",
				Sizes: []openrtb.Format{
					{
						W: 100,
						H: 100,
					},
				},
				Code: "test_adunitcode2",
				MediaTypes: []pbs.MediaType{
					pbs.MEDIA_TYPE_BANNER,
				},
			},
			{
				BidID: "test_bidid3",
				Sizes: []openrtb.Format{
					{
						W: 200,
						H: 200,
					},
				},
				Code: "test_adunitcode3",
				MediaTypes: []pbs.MediaType{
					pbs.MEDIA_TYPE_BANNER,
				},
			},
			{
				BidID: "test_bidid_video",
				Sizes: []openrtb.Format{
					{
						W: 400,
						H: 400,
					},
				},
				Code: "test_adunitcode_video",
				MediaTypes: []pbs.MediaType{
					pbs.MEDIA_TYPE_VIDEO,
				},
			},
			{
				BidID: "test_bidid3",
				Sizes: []openrtb.Format{
					{
						W: 150,
						H: 150,
					},
				},
				Code: "test_adunitcode_x",
				MediaTypes: []pbs.MediaType{
					pbs.MEDIA_TYPE_BANNER,
				},
			},
			{
				BidID: "test_bidid_y",
				Sizes: []openrtb.Format{
					{
						W: 150,
						H: 150,
					},
				},
				Code: "test_adunitcode_3",
				MediaTypes: []pbs.MediaType{
					pbs.MEDIA_TYPE_BANNER,
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

func TestNewJsonDirectoryServer(t *testing.T) {
	handler := NewJsonDirectoryServer(&testValidator{})
	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/whatever", nil)
	handler(recorder, request, nil)

	var data map[string]json.RawMessage
	json.Unmarshal(recorder.Body.Bytes(), &data)

	// Make sure that every adapter has a json schema by the same name associated with it.
	adapterFiles, err := ioutil.ReadDir(adapterDirectory)
	if err != nil {
		t.Fatalf("Failed to open the adapters directory: %v", err)
	}

	for _, adapterFile := range adapterFiles {
		if adapterFile.IsDir() && adapterFile.Name() != "adapterstest" {
			ensureHasKey(t, data, adapterFile.Name())
		}
	}
}

func TestWriteAuctionError(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeAuctionError(recorder, "some error message", nil)
	var resp pbs.PBSResponse
	json.Unmarshal(recorder.Body.Bytes(), &resp)

	if len(resp.Bids) != 0 {
		t.Error("Error responses should return no bids.")
	}
	if resp.Status != "some error message" {
		t.Errorf("The response status should be the error message. Got: %s", resp.Status)
	}

	if len(resp.BidderStatus) != 0 {
		t.Errorf("Error responses shouldn't have any BidderStatus elements. Got %d", len(resp.BidderStatus))
	}
}

func ensureHasKey(t *testing.T, data map[string]json.RawMessage, key string) {
	t.Helper()
	if _, ok := data[key]; !ok {
		t.Errorf("Expected map to produce a schema for adapter: %s", key)
	}
}

func TestNewFilesFetcher(t *testing.T) {
	fetcher, _, err := NewFetchers(&config.StoredRequests{
		Files: true,
	}, nil)
	if err != nil {
		t.Errorf("Error constructing file backends. %v", err)
	}
	if fetcher == nil {
		t.Errorf("The file-backed fetcher should be non-nil.")
	}
}

func TestNewEmptyFetcher(t *testing.T) {
	fetcher, _, err := NewFetchers(&config.StoredRequests{}, nil)
	if err != nil {
		t.Errorf("Error constructing backends. %v", err)
	}
	if fetcher == nil {
		t.Errorf("The fetcher should be non-nil, even with an empty config.")
	}
	if _, errs := fetcher.FetchRequests(context.Background(), []string{"some-id"}); len(errs) != 1 {
		t.Errorf("The returned accountFetcher should fail on any ID.")
	}
	if _, errs := fetcher.FetchRequests(context.Background(), []string{"some-id"}); len(errs) != 1 {
		t.Errorf("The returned requestFetcher should fail on any ID.")
	}
}

func TestExchangeMap(t *testing.T) {
	exchanges := newExchangeMap(&config.Configuration{})
	for bidderName := range exchanges {
		// OpenRTB doesn't support hardcoded aliases... so this test skips districtm,
		// which was the only alias in the legacy adapter map.
		if _, ok := openrtb_ext.BidderMap[bidderName]; bidderName != "districtm" && !ok {
			t.Errorf("Bidder %s exists in exchange, but is not a part of the BidderMap.", bidderName)
		}
	}
}

type testValidator struct{}

func (validator *testValidator) Validate(name openrtb_ext.BidderName, ext openrtb.RawJSON) error {
	return nil
}

func (validator *testValidator) Schema(name openrtb_ext.BidderName) string {
	if name == openrtb_ext.BidderAppnexus {
		return "{\"appnexus\":true}"
	} else {
		return "{\"appnexus\":false}"
	}
}

// Test the viper setup
func TestViperInit(t *testing.T) {
	compareStrings(t, "Viper error: external_url expected to be %s, found %s", "http://localhost:8000", viper.Get("external_url").(string))
	compareStrings(t, "Viper error: adapters.pulsepoint.endpoint expected to be %s, found %s", "http://bid.contextweb.com/header/s/ortb/prebid-s2s", viper.Get("adapters.pulsepoint.endpoint").(string))
}

func TestViperEnv(t *testing.T) {
	port := forceEnv(t, "PBS_PORT", "7777")
	defer port()

	endpt := forceEnv(t, "PBS_ADAPTERS_PUBMATIC_ENDPOINT", "not_an_endpoint")
	defer endpt()

	ttl := forceEnv(t, "PBS_HOST_COOKIE_TTL_DAYS", "60")
	defer ttl()

	// Basic config set
	compareStrings(t, "Viper error: port expected to be %s, found %s", "7777", viper.Get("port").(string))
	// Nested config set
	compareStrings(t, "Viper error: adapters.pubmatic.endpoint expected to be %s, found %s", "not_an_endpoint", viper.Get("adapters.pubmatic.endpoint").(string))
	// Config set with underscores
	compareStrings(t, "Viper error: host_cookie.ttl_days expected to be %s, found %s", "60", viper.Get("host_cookie.ttl_days").(string))
}

func compareStrings(t *testing.T, message string, expect string, actual string) {
	if expect != actual {
		t.Errorf(message, expect, actual)
	}
}

// forceEnv sets an environment variable to a certain value, and return a deferable function to reset it to the original value.
func forceEnv(t *testing.T, key string, val string) func() {
	orig, set := os.LookupEnv(key)
	err := os.Setenv(key, val)
	if err != nil {
		t.Fatalf("Error setting evnvironment %s", key)
	}
	if set {
		return func() {
			if os.Setenv(key, orig) != nil {
				t.Fatalf("Error unsetting evnvironment %s", key)
			}
		}
	} else {
		return func() {
			if os.Unsetenv(key) != nil {
				t.Fatalf("Error unsetting evnvironment %s", key)
			}
		}
	}
}
