package endpoints

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	metricsConf "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/prebid_cache_client"
	gdprPolicy "github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/prebid/prebid-server/usersync/usersyncers"
	"github.com/spf13/viper"

	"github.com/stretchr/testify/assert"
)

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
	hcc := config.HostCookie{}

	pbs_req, err := pbs.ParsePBSRequest(r, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, d, &hcc)
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
			if bid.AdServerTargeting[string(openrtb_ext.HbSizeConstantKey)] != "300x250" {
				t.Error(string(openrtb_ext.HbSizeConstantKey) + " key was not parsed correctly")
			}
			if bid.AdServerTargeting[string(openrtb_ext.HbpbConstantKey)] != "2.00" {
				t.Error(string(openrtb_ext.HbpbConstantKey)+" key was not parsed correctly ", bid.AdServerTargeting[string(openrtb_ext.HbpbConstantKey)])
			}

			if bid.AdServerTargeting[string(openrtb_ext.HbCacheKey)] != "test_cache_id1" {
				t.Error(string(openrtb_ext.HbCacheKey) + " key was not parsed correctly")
			}
			if bid.AdServerTargeting[string(openrtb_ext.HbBidderConstantKey)] != "audienceNetwork" {
				t.Error(string(openrtb_ext.HbBidderConstantKey) + " key was not parsed correctly")
			}
			if bid.AdServerTargeting[string(openrtb_ext.HbDealIDConstantKey)] != "2345" {
				t.Error(string(openrtb_ext.HbDealIDConstantKey) + " key was not parsed correctly ")
			}
		}
		if bid.BidderCode == "appnexus" {
			if bid.AdServerTargeting[string(openrtb_ext.HbSizeConstantKey)+"_appnexus"] != "320x50" {
				t.Error(string(openrtb_ext.HbSizeConstantKey) + " key for appnexus bidder was not parsed correctly")
			}
			if bid.AdServerTargeting[string(openrtb_ext.HbCacheKey)+"_appnexus"] != "test_cache_id2" {
				t.Error(string(openrtb_ext.HbCacheKey) + " key for appnexus bidder was not parsed correctly")
			}
			if bid.AdServerTargeting[string(openrtb_ext.HbBidderConstantKey)+"_appnexus"] != "appnexus" {
				t.Error(string(openrtb_ext.HbBidderConstantKey) + " key for appnexus bidder was not parsed correctly")
			}
			if bid.AdServerTargeting[string(openrtb_ext.HbpbConstantKey)+"_appnexus"] != "1.00" {
				t.Error(string(openrtb_ext.HbpbConstantKey) + " key for appnexus bidder was not parsed correctly")
			}
			if bid.AdServerTargeting[string(openrtb_ext.HbpbConstantKey)] != "" {
				t.Error(string(openrtb_ext.HbpbConstantKey) + " key was parsed for two bidders")
			}
			if bid.AdServerTargeting[string(openrtb_ext.HbDealIDConstantKey)+"_appnexus"] != "1234" {
				t.Errorf(string(openrtb_ext.HbDealIDConstantKey)+"_appnexus was not parsed correctly %v", bid.AdServerTargeting[string(openrtb_ext.HbDealIDConstantKey)+"_appnexus"])
			}
		}
		if bid.BidderCode == string(openrtb_ext.BidderRubicon) {
			if bid.AdServerTargeting["rpfl_1001"] != "15_tier0100" {
				t.Error("custom ad_server_targeting KVPs from adapter were not preserved")
			}
		}
		if bid.BidderCode == "nosizebidder" {
			if _, exists := bid.AdServerTargeting[string(openrtb_ext.HbSizeConstantKey)+"_nosizebidder"]; exists {
				t.Error(string(openrtb_ext.HbSizeConstantKey)+" key for nosize bidder was not parsed correctly", bid.AdServerTargeting)
			}
		}
		if bid.BidderCode == "nodeal" {
			if _, exists := bid.AdServerTargeting[string(openrtb_ext.HbDealIDConstantKey)+"_nodeal"]; exists {
				t.Error(string(openrtb_ext.HbDealIDConstantKey) + " key for nodeal bidder was not parsed correctly")
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

	b, err := json.Marshal(&resp)
	if err != nil {
		http.Error(w, "Failed to serialize UUIDs into JSON.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
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
	v := viper.New()
	config.SetupViper(v, "")
	v.Set("gdpr.default_value", "0")
	cfg, err := config.New(v)
	if err != nil {
		t.Fatal(err.Error())
	}
	syncers := usersyncers.NewSyncerMap(cfg)
	gdprPerms := gdpr.NewPermissions(context.Background(), config.GDPR{
		HostVendorID: 0,
	}, nil, nil)
	prebid_cache_client.InitPrebidCache(server.URL)
	var labels = &metrics.Labels{}
	if err := cacheVideoOnly(bids, ctx, &auction{cfg: cfg, syncers: syncers, gdprPerms: gdprPerms, metricsEngine: &metricsConf.DummyMetricsEngine{}}, labels); err != nil {
		t.Errorf("Prebid cache failed: %v \n", err)
		return
	}
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

func TestShouldUsersync(t *testing.T) {
	tests := []struct {
		description      string
		signal           string
		allowHostCookies bool
		allowBidderSync  bool
		wantAllow        bool
	}{
		{
			description:      "Don't sync - GDPR on, host cookies disallows and bidder sync disallows",
			signal:           "1",
			allowHostCookies: false,
			allowBidderSync:  false,
			wantAllow:        false,
		},
		{
			description:      "Don't sync - GDPR on, host cookies disallows and bidder sync allows",
			signal:           "1",
			allowHostCookies: false,
			allowBidderSync:  true,
			wantAllow:        false,
		},
		{
			description:      "Don't sync - GDPR on, host cookies allows and bidder sync disallows",
			signal:           "1",
			allowHostCookies: true,
			allowBidderSync:  false,
			wantAllow:        false,
		},
		{
			description:      "Sync - GDPR on, host cookies allows and bidder sync allows",
			signal:           "1",
			allowHostCookies: true,
			allowBidderSync:  true,
			wantAllow:        true,
		},
		{
			description:      "Don't sync - invalid GDPR signal, host cookies disallows and bidder sync disallows",
			signal:           "2",
			allowHostCookies: false,
			allowBidderSync:  false,
			wantAllow:        false,
		},
	}

	for _, tt := range tests {
		deps := auction{
			gdprPerms: &auctionMockPermissions{
				allowBidderSync:  tt.allowBidderSync,
				allowHostCookies: tt.allowHostCookies,
			},
		}
		gdprPrivacyPolicy := gdprPolicy.Policy{
			Signal: tt.signal,
		}

		allow := deps.shouldUsersync(context.Background(), openrtb_ext.BidderAdform, gdprPrivacyPolicy)
		assert.Equal(t, tt.wantAllow, allow, tt.description)
	}
}

type auctionMockPermissions struct {
	allowBidderSync  bool
	allowHostCookies bool
	allowBidRequest  bool
	passGeo          bool
	passID           bool
}

func (m *auctionMockPermissions) HostCookiesAllowed(ctx context.Context, gdprSignal gdpr.Signal, consent string) (bool, error) {
	return m.allowHostCookies, nil
}

func (m *auctionMockPermissions) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, gdprSignal gdpr.Signal, consent string) (bool, error) {
	return m.allowBidderSync, nil
}

func (m *auctionMockPermissions) AuctionActivitiesAllowed(ctx context.Context, bidder openrtb_ext.BidderName, PublisherID string, gdprSignal gdpr.Signal, consent string, weakVendorEnforcement bool) (allowBidRequest bool, passGeo bool, passID bool, err error) {
	return m.allowBidRequest, m.passGeo, m.passID, nil
}

func TestBidSizeValidate(t *testing.T) {
	bids := make(pbs.PBSBidSlice, 0)
	// bid1 will be rejected due to undefined size when adunit has multiple sizes
	bid1 := pbs.PBSBid{
		BidID:      "test_bidid1",
		AdUnitCode: "test_adunitcode1",
		BidderCode: "randNetwork",
		Price:      1.05,
		Adm:        "test_adm",
		// Width:             100,
		// Height:            100,
		CreativeMediaType: "banner",
	}
	bids = append(bids, &bid1)
	// bid2 will be considered a normal ideal banner bid
	bid2 := pbs.PBSBid{
		BidID:             "test_bidid2",
		AdUnitCode:        "test_adunitcode2",
		BidderCode:        "randNetwork",
		Price:             1.05,
		Adm:               "test_adm",
		Width:             100,
		Height:            100,
		CreativeMediaType: "banner",
	}
	bids = append(bids, &bid2)
	// bid3 will have it's dimensions set based on sizes defined in request
	bid3 := pbs.PBSBid{
		BidID:      "test_bidid3",
		AdUnitCode: "test_adunitcode3",
		BidderCode: "randNetwork",
		Price:      1.05,
		Adm:        "test_adm",
		//Width:             200,
		//Height:            200,
		CreativeMediaType: "banner",
	}

	bids = append(bids, &bid3)

	// bid4 will be ignored as it's a video creative type
	bid4 := pbs.PBSBid{
		BidID:      "test_bidid_video",
		AdUnitCode: "test_adunitcode_video",
		BidderCode: "randNetwork",
		Price:      1.05,
		Adm:        "test_adm",
		//Width:             400,
		//Height:            400,
		CreativeMediaType: "video",
	}

	bids = append(bids, &bid4)

	mybidder := pbs.PBSBidder{
		BidderCode: "randNetwork",
		AdUnitCode: "test_adunitcode",
		AdUnits: []pbs.PBSAdUnit{
			{
				BidID: "test_bidid1",
				Sizes: []openrtb2.Format{
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
				Sizes: []openrtb2.Format{
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
				Sizes: []openrtb2.Format{
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
				Sizes: []openrtb2.Format{
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
				Sizes: []openrtb2.Format{
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
				Sizes: []openrtb2.Format{
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

func TestPanicRecovery(t *testing.T) {
	dummy := auction{
		cfg:     nil,
		syncers: nil,
		gdprPerms: &auctionMockPermissions{
			allowBidderSync:  false,
			allowHostCookies: false,
		},
		metricsEngine: &metricsConf.DummyMetricsEngine{},
	}
	panicker := func(bidder *pbs.PBSBidder, blables metrics.AdapterLabels) {
		panic("panic!")
	}
	recovered := dummy.recoverSafely(panicker)
	recovered(nil, metrics.AdapterLabels{})
}
