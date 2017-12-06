package rubicon

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/pbs"

	"fmt"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
)

type rubiAppendTrackerUrlTestScenario struct {
	source   string
	tracker  string
	expected string
}

type rubiTagInfo struct {
	code              string
	zoneID            int
	bid               float64
	content           string
	adServerTargeting map[string]string
	mediaType         string
}

type rubiBidInfo struct {
	domain             string
	page               string
	accountID          int
	siteID             int
	tags               []rubiTagInfo
	deviceIP           string
	deviceUA           string
	buyerUID           string
	xapiuser           string
	xapipass           string
	delay              time.Duration
	visitorTargeting   string
	inventoryTargeting string
	sdkVersion         string
	sdkPlatform        string
	sdkSource          string
	devicePxRatio      float64
}

var rubidata rubiBidInfo

func DummyRubiconServer(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println("Request", string(body))

	var breq openrtb.BidRequest
	err = json.Unmarshal(body, &breq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(breq.Imp) > 1 {
		http.Error(w, "Rubicon adapter only supports one Imp per request", http.StatusInternalServerError)
		return
	}
	imp := breq.Imp[0]
	var rix rubiconImpExt
	err = json.Unmarshal(imp.Ext, &rix)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	impTargetingString, _ := json.Marshal(&rix.RP.Target)
	if string(impTargetingString) != rubidata.inventoryTargeting {
		http.Error(w, fmt.Sprintf("Inventory FPD targeting '%s' doesn't match '%s'", string(impTargetingString), rubidata.inventoryTargeting), http.StatusInternalServerError)
		return
	}
	if rix.RP.Track.Mint != "prebid" {
		http.Error(w, fmt.Sprintf("Track mint '%s' doesn't match '%s'", rix.RP.Track.Mint, "prebid"), http.StatusInternalServerError)
		return
	}
	mintVersionString := rubidata.sdkSource + "_" + rubidata.sdkPlatform + "_" + rubidata.sdkVersion
	if rix.RP.Track.MintVersion != mintVersionString {
		http.Error(w, fmt.Sprintf("Track mint version '%s' doesn't match '%s'", rix.RP.Track.MintVersion, mintVersionString), http.StatusInternalServerError)
		return
	}

	ix := -1

	for i, tag := range rubidata.tags {
		if rix.RP.ZoneID == tag.zoneID {
			ix = i
		}
	}
	if ix == -1 {
		http.Error(w, fmt.Sprintf("Zone %d not found", rix.RP.ZoneID), http.StatusInternalServerError)
		return
	}

	resp := openrtb.BidResponse{
		ID:    "test-response-id",
		BidID: "test-bid-id",
		Cur:   "USD",
		SeatBid: []openrtb.SeatBid{
			{
				Seat: "RUBICON",
				Bid:  make([]openrtb.Bid, 2),
			},
		},
	}

	if imp.Banner != nil {
		var bix rubiconBannerExt
		err = json.Unmarshal(imp.Banner.Ext, &bix)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if bix.RP.SizeID != 10 { // 300x600
			http.Error(w, fmt.Sprintf("Primary size ID isn't 10"), http.StatusInternalServerError)
			return
		}
		if len(bix.RP.AltSizeIDs) != 1 || bix.RP.AltSizeIDs[0] != 15 { // 300x250
			http.Error(w, fmt.Sprintf("Alt size ID isn't 15"), http.StatusInternalServerError)
			return
		}
		if bix.RP.MIME != "text/html" {
			http.Error(w, fmt.Sprintf("MIME isn't text/html"), http.StatusInternalServerError)
			return
		}
	}

	if imp.Video != nil {
		var vix rubiconVideoExt
		err = json.Unmarshal(imp.Video.Ext, &vix)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(imp.Video.MIMEs) == 0 {
			http.Error(w, fmt.Sprintf("Empty imp.video.mimes array"), http.StatusInternalServerError)
			return
		}
		if len(imp.Video.Protocols) == 0 {
			http.Error(w, fmt.Sprintf("Empty imp.video.protocols array"), http.StatusInternalServerError)
			return
		}
		for _, protocol := range imp.Video.Protocols {
			if protocol < 1 || protocol > 8 {
				http.Error(w, fmt.Sprintf("Invalid video protocol %d", protocol), http.StatusInternalServerError)
				return
			}
		}
	}

	targeting := "{\"rp\":{\"targeting\":[{\"key\":\"key1\",\"values\":[\"value1\"]},{\"key\":\"key2\",\"values\":[\"value2\"]}]}}"
	rawTargeting := openrtb.RawJSON(targeting)

	resp.SeatBid[0].Bid[0] = openrtb.Bid{
		ID:    "random-id",
		ImpID: imp.ID,
		Price: rubidata.tags[ix].bid,
		AdM:   rubidata.tags[ix].content,
		Ext:   rawTargeting,
	}

	if breq.Site == nil {
		http.Error(w, fmt.Sprintf("No site object sent"), http.StatusInternalServerError)
		return
	}
	if breq.Site.Domain != rubidata.domain {
		http.Error(w, fmt.Sprintf("Domain '%s' doesn't match '%s", breq.Site.Domain, rubidata.domain), http.StatusInternalServerError)
		return
	}
	if breq.Site.Page != rubidata.page {
		http.Error(w, fmt.Sprintf("Page '%s' doesn't match '%s", breq.Site.Page, rubidata.page), http.StatusInternalServerError)
		return
	}
	var rsx rubiconSiteExt
	err = json.Unmarshal(breq.Site.Ext, &rsx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rsx.RP.SiteID != rubidata.siteID {
		http.Error(w, fmt.Sprintf("SiteID '%d' doesn't match '%d", rsx.RP.SiteID, rubidata.siteID), http.StatusInternalServerError)
		return
	}
	if breq.Site.Publisher == nil {
		http.Error(w, fmt.Sprintf("No site.publisher object sent"), http.StatusInternalServerError)
		return
	}
	var rpx rubiconPubExt
	err = json.Unmarshal(breq.Site.Publisher.Ext, &rpx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rpx.RP.AccountID != rubidata.accountID {
		http.Error(w, fmt.Sprintf("AccountID '%d' doesn't match '%d'", rpx.RP.AccountID, rubidata.accountID), http.StatusInternalServerError)
		return
	}
	if breq.Device.UA != rubidata.deviceUA {
		http.Error(w, fmt.Sprintf("UA '%s' doesn't match '%s'", breq.Device.UA, rubidata.deviceUA), http.StatusInternalServerError)
		return
	}
	if breq.Device.IP != rubidata.deviceIP {
		http.Error(w, fmt.Sprintf("IP '%s' doesn't match '%s'", breq.Device.IP, rubidata.deviceIP), http.StatusInternalServerError)
		return
	}
	if breq.Device.PxRatio != rubidata.devicePxRatio {
		http.Error(w, fmt.Sprintf("Pixel ratio '%f' doesn't match '%f'", breq.Device.PxRatio, rubidata.devicePxRatio), http.StatusInternalServerError)
		return
	}
	if breq.User.BuyerUID != rubidata.buyerUID {
		http.Error(w, fmt.Sprintf("User ID '%s' doesn't match '%s'", breq.User.BuyerUID, rubidata.buyerUID), http.StatusInternalServerError)
		return
	}

	var rux rubiconUserExt
	err = json.Unmarshal(breq.User.Ext, &rux)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userTargetingString, _ := json.Marshal(&rux.RP.Target)
	if string(userTargetingString) != rubidata.visitorTargeting {
		http.Error(w, fmt.Sprintf("User FPD targeting '%s' doesn't match '%s'", string(userTargetingString), rubidata.visitorTargeting), http.StatusInternalServerError)
		return
	}

	if rubidata.delay > 0 {
		<-time.After(rubidata.delay)
	}

	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func TestRubiconBasicResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyRubiconServer))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)

	bids, err := an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}
	if len(bids) != 3 {
		t.Fatalf("Received %d bids instead of 3", len(bids))
	}
	for _, bid := range bids {
		matched := false
		for _, tag := range rubidata.tags {
			if bid.AdUnitCode == tag.code {
				matched = true
				if bid.BidderCode != "rubicon" {
					t.Errorf("Incorrect BidderCode '%s'", bid.BidderCode)
				}
				if bid.Price != tag.bid {
					t.Errorf("Incorrect bid price '%.2f' expected '%.2f'", bid.Price, tag.bid)
				}
				if bid.Adm != tag.content {
					t.Errorf("Incorrect bid markup '%s' expected '%s'", bid.Adm, tag.content)
				}
				if !reflect.DeepEqual(bid.AdServerTargeting, tag.adServerTargeting) {
					t.Errorf("Incorrect targeting '%+v' expected '%+v'", bid.AdServerTargeting, tag.adServerTargeting)
				}
			}
		}
		if !matched {
			t.Errorf("Received bid for unknown ad unit '%s'", bid.AdUnitCode)
		}
	}

	// test debug enabled
	pbReq.IsDebug = true
	_, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}

	// same test but with request timing out
	rubidata.delay = 20 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	bids, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err == nil {
		t.Fatalf("Should have gotten a timeout error: %v", err)
	}
}

func TestRubiconUserSyncInfo(t *testing.T) {
	url := "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid"

	an := NewRubiconAdapter(adapters.DefaultHTTPAdapterConfig, "uri", "xuser", "xpass", "pbs-test-tracker", url)
	if an.usersyncInfo.URL != url {
		t.Fatalf("should have matched")
	}
	if an.usersyncInfo.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if an.usersyncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}

	name := an.Name()
	if name != "Rubicon" {
		t.Errorf("Name '%s' != 'Rubicon'", name)
	}

	familyName := an.FamilyName()
	if familyName != "rubicon" {
		t.Errorf("FamilyName '%s' != 'rubicon'", familyName)
	}

	skipNoCookies := an.SkipNoCookies()
	if skipNoCookies != false {
		t.Errorf("SkipNoCookies should be false")
	}

	usersyncInfo := an.GetUsersyncInfo()
	if usersyncInfo.URL != url {
		t.Fatalf("URL '%s' != '%s'", usersyncInfo.URL, url)
	}
	if usersyncInfo.Type != "redirect" {
		t.Fatalf("Type should be redirect")
	}
	if usersyncInfo.SupportCORS != false {
		t.Fatalf("SupportCORS should be false")
	}

}

func TestParseSizes(t *testing.T) {
	sizes := []openrtb.Format{
		{
			W: 300,
			H: 600,
		},
		{
			W: 300,
			H: 250,
		},
	}
	primary, alt, err := parseRubiconSizes(sizes)
	if err != nil {
		t.Errorf("Parsing error: %v", err)
	}
	if primary != 10 {
		t.Errorf("Primary %d != 10", primary)
	}
	if len(alt) != 1 {
		t.Fatalf("Alt not len 1")
	}
	if alt[0] != 15 {
		t.Errorf("Alt not 15: %d", alt[0])
	}

	sizes = []openrtb.Format{
		{
			W: 1111,
			H: 1111,
		},
		{
			W: 300,
			H: 250,
		},
	}
	primary, alt, err = parseRubiconSizes(sizes)

	if err != nil {
		t.Errorf("Shouldn't have thrown error for invalid size 1111x1111 since we still have a valid one")
	}
	if primary != 15 {
		t.Errorf("Primary %d != 15", primary)
	}
	if len(alt) != 0 {
		t.Fatalf("Alt len %d != 0", len(alt))
	}

	sizes = []openrtb.Format{
		{
			W: 300,
			H: 250,
		},
	}
	primary, alt, err = parseRubiconSizes(sizes)

	if err != nil {
		t.Errorf("Parsing error: %v", err)
	}
	if primary != 15 {
		t.Errorf("Primary %d != 15", primary)
	}
	if len(alt) != 0 {
		t.Fatalf("Alt len %d != 0", len(alt))
	}

	sizes = []openrtb.Format{
		{
			W: 123,
			H: 456,
		},
	}
	primary, alt, err = parseRubiconSizes(sizes)

	if err == nil {
		t.Errorf("Parsing error: %v", err)
	}
	if primary != 0 {
		t.Errorf("Primary %d != 0", primary)
	}
	if len(alt) != 0 {
		t.Errorf("Alt len %d != 0", len(alt))
	}
}

func TestAppendTracker(t *testing.T) {
	testScenarios := []rubiAppendTrackerUrlTestScenario{
		{
			source:   "http://test.url/",
			tracker:  "prebid",
			expected: "http://test.url/?tk_xint=prebid",
		},
		{
			source:   "http://test.url/?hello=true",
			tracker:  "prebid",
			expected: "http://test.url/?hello=true&tk_xint=prebid",
		},
	}

	for _, scenario := range testScenarios {
		res := appendTrackerToUrl(scenario.source, scenario.tracker)
		if res != scenario.expected {
			t.Fatalf("Failed to convert '%s' to '%s'", res, scenario.expected)
		}
	}
}

func TestNoContentResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	_, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}

}

func TestNotFoundResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	_, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	if err == nil {
		t.Fatalf("Should have gotten an error: %v", err)
	}

}

func TestWrongFormatResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This is text."))
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	_, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	if err == nil {
		t.Fatalf("Should have gotten an error: %v", err)
	}

}

func TestZeroSeatBidResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openrtb.BidResponse{
			ID:      "test-response-id",
			BidID:   "test-bid-id",
			Cur:     "USD",
			SeatBid: []openrtb.SeatBid{},
		}
		js, _ := json.Marshal(resp)
		w.Write(js)
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	_, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}

}

func TestEmptyBidResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openrtb.BidResponse{
			ID:    "test-response-id",
			BidID: "test-bid-id",
			Cur:   "USD",
			SeatBid: []openrtb.SeatBid{
				{
					Seat: "RUBICON",
					Bid:  make([]openrtb.Bid, 0),
				},
			},
		}
		js, _ := json.Marshal(resp)
		w.Write(js)
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	_, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}

}

func TestWrongBidIdResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openrtb.BidResponse{
			ID:    "test-response-id",
			BidID: "test-bid-id",
			Cur:   "USD",
			SeatBid: []openrtb.SeatBid{
				{
					Seat: "RUBICON",
					Bid:  make([]openrtb.Bid, 2),
				},
			},
		}
		resp.SeatBid[0].Bid[0] = openrtb.Bid{
			ID:    "random-id",
			ImpID: "zma",
			Price: 1.67,
			AdM:   "zma",
			Ext:   openrtb.RawJSON("{\"rp\":{\"targeting\":[{\"key\":\"key1\",\"values\":[\"value1\"]},{\"key\":\"key2\",\"values\":[\"value2\"]}]}}"),
		}
		js, _ := json.Marshal(resp)
		w.Write(js)
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	_, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	if err == nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}

}

func TestDifferentRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyRubiconServer))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)

	// test app not nil
	pbReq.App = &openrtb.App{
		ID:   "com.test",
		Name: "testApp",
	}
	_, err := an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err == nil {
		t.Fatalf("Should have gotten an error: %v", err)
	}

	// set app back to normal
	pbReq.App = nil

	// test video media type
	pbReq.Bidders[0].AdUnits[0].MediaTypes = []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO}
	pbReq.Bidders[0].AdUnits[1].MediaTypes = []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO}
	_, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err == nil {
		t.Fatalf("Should have gotten an error: %v", err)
	}

	// set media back to normal
	pbReq.Bidders[0].AdUnits[0].MediaTypes = []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}
	pbReq.Bidders[0].AdUnits[1].MediaTypes = []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}

	// test wrong params
	pbReq.Bidders[0].AdUnits[0].Params = json.RawMessage(fmt.Sprintf("{\"zoneId\": %s, \"siteId\": %d, \"visitor\": %s, \"inventory\": %s}", "zma", rubidata.siteID, rubidata.visitorTargeting, rubidata.inventoryTargeting))
	_, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err == nil {
		t.Fatalf("Should have gotten an error: %v", err)
	}

	// test no account id
	pbReq.Bidders[0].AdUnits[0].Params = json.RawMessage(fmt.Sprintf("{\"zoneId\": %d, \"siteId\": %d, \"visitor\": %s, \"inventory\": %s}", 8394, rubidata.siteID, rubidata.visitorTargeting, rubidata.inventoryTargeting))
	_, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err == nil {
		t.Fatalf("Should have gotten an error: %v", err)
	}

	// test no site id
	pbReq.Bidders[0].AdUnits[0].Params = json.RawMessage(fmt.Sprintf("{\"zoneId\": %d, \"accountId\": %d, \"visitor\": %s, \"inventory\": %s}", 8394, rubidata.accountID, rubidata.visitorTargeting, rubidata.inventoryTargeting))
	_, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err == nil {
		t.Fatalf("Should have gotten an error: %v", err)
	}

	// test no zone id
	pbReq.Bidders[0].AdUnits[0].Params = json.RawMessage(fmt.Sprintf("{\"siteId\": %d, \"accountId\": %d, \"visitor\": %s, \"inventory\": %s}", rubidata.siteID, rubidata.accountID, rubidata.visitorTargeting, rubidata.inventoryTargeting))
	_, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err == nil {
		t.Fatalf("Should have gotten an error: %v", err)
	}

	// set params back to normal
	pbReq.Bidders[0].AdUnits[0].Params = json.RawMessage(fmt.Sprintf("{\"zoneId\": %d, \"siteId\": %d, \"accountId\": %d, \"visitor\": %s, \"inventory\": %s}", 8394, rubidata.siteID, rubidata.accountID, rubidata.visitorTargeting, rubidata.inventoryTargeting))

	// test invalid size
	pbReq.Bidders[0].AdUnits[0].Sizes = []openrtb.Format{
		{
			W: 0,
			H: 0,
		},
	}
	_, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err == nil {
		t.Fatalf("Should have gotten an error: %v", err)
	}

}

func CreatePrebidRequest(server *httptest.Server, t *testing.T) (an *RubiconAdapter, ctx context.Context, pbReq *pbs.PBSRequest) {
	rubidata = rubiBidInfo{
		domain:             "nytimes.com",
		page:               "https://www.nytimes.com/2017/05/04/movies/guardians-of-the-galaxy-2-review-chris-pratt.html?hpw&rref=movies&action=click&pgtype=Homepage&module=well-region&region=bottom-well&WT.nav=bottom-well&_r=0",
		accountID:          7891,
		siteID:             283282,
		tags:               make([]rubiTagInfo, 3),
		deviceIP:           "25.91.96.36",
		deviceUA:           "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_4) AppleWebKit/603.1.30 (KHTML, like Gecko) Version/10.1 Safari/603.1.30",
		buyerUID:           "need-an-actual-rp-id",
		visitorTargeting:   "[\"v1\",\"v2\"]",
		inventoryTargeting: "[\"i1\",\"i2\"]",
		sdkVersion:         "2.0.0",
		sdkPlatform:        "iOS",
		sdkSource:          "some-sdk",
		devicePxRatio:      4.0,
	}

	targeting := make(map[string]string, 2)
	targeting["key1"] = "value1"
	targeting["key2"] = "value2"

	rubidata.tags[0] = rubiTagInfo{
		code:              "first-tag",
		zoneID:            8394,
		bid:               1.67,
		adServerTargeting: targeting,
		mediaType:         "banner",
	}
	rubidata.tags[1] = rubiTagInfo{
		code:              "second-tag",
		zoneID:            8395,
		bid:               3.22,
		adServerTargeting: targeting,
		mediaType:         "banner",
	}
	rubidata.tags[2] = rubiTagInfo{
		code:              "video-tag",
		zoneID:            7780,
		bid:               23.12,
		adServerTargeting: targeting,
		mediaType:         "video",
	}

	conf := *adapters.DefaultHTTPAdapterConfig
	an = NewRubiconAdapter(&conf, "uri", rubidata.xapiuser, rubidata.xapipass, "pbs-test-tracker", "localhost/usersync")
	an.URI = server.URL

	pbin := pbs.PBSRequest{
		AdUnits: make([]pbs.AdUnit, 3),
		Device:  &openrtb.Device{PxRatio: rubidata.devicePxRatio},
		SDK:     &pbs.SDK{Source: rubidata.sdkSource, Platform: rubidata.sdkPlatform, Version: rubidata.sdkVersion},
	}

	for i, tag := range rubidata.tags {
		pbin.AdUnits[i] = pbs.AdUnit{
			Code:       tag.code,
			MediaTypes: []string{tag.mediaType},
			Sizes: []openrtb.Format{
				{
					W: 300,
					H: 600,
				},
				{
					W: 300,
					H: 250,
				},
			},
			Bids: []pbs.Bids{
				{
					BidderCode: "rubicon",
					BidID:      fmt.Sprintf("random-id-from-pbjs-%d", i),
					Params:     json.RawMessage(fmt.Sprintf("{\"zoneId\": %d, \"siteId\": %d, \"accountId\": %d, \"visitor\": %s, \"inventory\": %s}", tag.zoneID, rubidata.siteID, rubidata.accountID, rubidata.visitorTargeting, rubidata.inventoryTargeting)),
				},
			},
		}
		if tag.mediaType == "video" {
			pbin.AdUnits[i].Video = pbs.PBSVideo{
				Mimes:          []string{"video/mp4"},
				Minduration:    15,
				Maxduration:    30,
				Startdelay:     5,
				Skippable:      0,
				PlaybackMethod: 1,
				Protocols:      []int8{1, 2, 3, 4, 5},
			}
		}
	}

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(pbin)
	if err != nil {
		t.Fatalf("Json encoding failed: %v", err)
	}

	fmt.Println("body", body)

	req := httptest.NewRequest("POST", server.URL, body)
	req.Header.Add("Referer", rubidata.page)
	req.Header.Add("User-Agent", rubidata.deviceUA)
	req.Header.Add("X-Real-IP", rubidata.deviceIP)

	pc := pbs.ParsePBSCookieFromRequest(req, &config.Cookie{})
	pc.TrySync("rubicon", rubidata.buyerUID)
	fakewriter := httptest.NewRecorder()
	pc.SetCookieOnResponse(fakewriter, "")
	req.Header.Add("Cookie", fakewriter.Header().Get("Set-Cookie"))

	cacheClient, _ := dummycache.New()
	hcs := pbs.HostCookieSettings{}

	pbReq, err = pbs.ParsePBSRequest(req, cacheClient, &hcs)
	if err != nil {
		t.Fatalf("ParsePBSRequest failed: %v", err)
	}
	if len(pbReq.Bidders) != 1 {
		t.Fatalf("ParsePBSRequest returned %d bidders instead of 1", len(pbReq.Bidders))
	}
	if pbReq.Bidders[0].BidderCode != "rubicon" {
		t.Fatalf("ParsePBSRequest returned invalid bidder")
	}

	ctx = context.TODO()

	return

}
