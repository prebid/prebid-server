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

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/usersync"

	"fmt"

	"strings"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
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
		if bix.RP.SizeID != 15 { // 300x250
			http.Error(w, fmt.Sprintf("Primary size ID isn't 15"), http.StatusInternalServerError)
			return
		}
		if len(bix.RP.AltSizeIDs) != 1 || bix.RP.AltSizeIDs[0] != 10 { // 300x600
			http.Error(w, fmt.Sprintf("Alt size ID isn't 10"), http.StatusInternalServerError)
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
	rawTargeting := json.RawMessage(targeting)

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
				if bid.CreativeMediaType != tag.mediaType {
					t.Errorf("Incorrect media type '%s' expected '%s'", bid.CreativeMediaType, tag.mediaType)
				}
			}
		}
		if !matched {
			t.Errorf("Received bid for unknown ad unit '%s'", bid.AdUnitCode)
		}
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
	an := NewRubiconAdapter(adapters.DefaultHTTPAdapterConfig, "uri", "xuser", "xpass", "pbs-test-tracker")

	adapterName := an.Name()
	if adapterName != "rubicon" {
		t.Errorf("Name '%s' != 'rubicon'", adapterName)
	}

	skipNoCookies := an.SkipNoCookies()
	if skipNoCookies != false {
		t.Errorf("SkipNoCookies should be false")
	}
}

func getTestSizes() map[int]openrtb.Format {
	return map[int]openrtb.Format{
		15: {W: 300, H: 250},
		10: {W: 300, H: 600},
		2:  {W: 728, H: 91},
		9:  {W: 160, H: 600},
		8:  {W: 120, H: 600},
		33: {W: 180, H: 500},
		43: {W: 320, H: 50},
	}
}

func TestParseSizes(t *testing.T) {

	SIZE_ID := getTestSizes()

	sizes := []openrtb.Format{
		SIZE_ID[10],
		SIZE_ID[15],
	}
	primary, alt, err := parseRubiconSizes(sizes)
	if err != nil {
		t.Errorf("Parsing error: %v", err)
	}
	if primary != 15 {
		t.Errorf("Primary %d != 15", primary)
	}
	if len(alt) != 1 {
		t.Fatalf("Alt not len 1")
	}
	if alt[0] != 10 {
		t.Errorf("Alt not 10: %d", alt[0])
	}

	sizes = []openrtb.Format{
		{
			W: 1111,
			H: 2222,
		},
		SIZE_ID[15],
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
		SIZE_ID[15],
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
			W: 1111,
			H: 1222,
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

func TestMASAlgorithm(t *testing.T) {
	SIZE_ID := getTestSizes()
	type output struct {
		primary int
		alt     []int
		ok      bool
	}
	type testStub struct {
		input  []openrtb.Format
		output output
	}

	testStubs := []testStub{
		{
			[]openrtb.Format{
				SIZE_ID[2],
				SIZE_ID[9],
			},
			output{2, []int{9}, false},
		},
		{
			[]openrtb.Format{

				SIZE_ID[9],
				SIZE_ID[15],
			},
			output{15, []int{9}, false},
		},
		{
			[]openrtb.Format{
				SIZE_ID[2],
				SIZE_ID[15],
			},
			output{15, []int{2}, false},
		},
		{
			[]openrtb.Format{
				SIZE_ID[15],
				SIZE_ID[9],
				SIZE_ID[2],
			},
			output{15, []int{2, 9}, false},
		},
		{
			[]openrtb.Format{
				SIZE_ID[10],
				SIZE_ID[9],
				SIZE_ID[2],
			},
			output{2, []int{10, 9}, false},
		},
		{
			[]openrtb.Format{
				SIZE_ID[33],
				SIZE_ID[8],
				SIZE_ID[15],
			},
			output{15, []int{33, 8}, false},
		},
		{
			[]openrtb.Format{
				SIZE_ID[33],
				SIZE_ID[8],
				SIZE_ID[9],
				SIZE_ID[2],
			},
			output{2, []int{33, 8, 9}, false},
		},
		{
			[]openrtb.Format{
				SIZE_ID[33],
				SIZE_ID[8],
				SIZE_ID[9],
			},
			output{9, []int{33, 8}, false},
		},
		{
			[]openrtb.Format{
				SIZE_ID[33],
				SIZE_ID[8],
				SIZE_ID[2],
			},
			output{2, []int{33, 8}, false},
		},
		{
			[]openrtb.Format{
				SIZE_ID[33],
				SIZE_ID[2],
			},
			output{2, []int{33}, false},
		},
		{
			[]openrtb.Format{
				SIZE_ID[8],
			},
			output{8, []int{}, false},
		},
		{
			[]openrtb.Format{},
			output{0, []int{}, true},
		},
		{
			[]openrtb.Format{
				{W: 1111,
					H: 2345,
				},
			},
			output{0, []int{}, true},
		},
	}

	for _, test := range testStubs {
		prim, alt, err := parseRubiconSizes(test.input)
		if prim != test.output.primary {
			t.Fatalf("Error in parsing rubicon sizes: MAS algorithm fail at primary: testcase %v", test.input)
		}
		if len(alt) != len(test.output.alt) {
			t.Fatalf("Error in parsing rubicon sizes: MAS Algorithm fail at alt: testcase %v", test.input)
		}
		if (err != nil && !test.output.ok) || (err == nil && test.output.ok) {
			t.Fatalf("Error in parsing rubicon sizes: MAS Algorithm fail at throwing error: testcase %v", test.input)
		}
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
	bids, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	if len(bids) != 0 {
		t.Fatalf("Length of bids should be 0 instead of: %v", len(bids))
	}

	if pbReq.Bidders[0].Debug[0].StatusCode != 204 {
		t.Fatalf("StatusCode should be 204 instead of: %v", pbReq.Bidders[0].Debug[0].StatusCode)
	}

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

	if pbReq.Bidders[0].Debug[0].StatusCode != 404 {
		t.Fatalf("StatusCode should be 404 instead of: %v", pbReq.Bidders[0].Debug[0].StatusCode)
	}

	if err == nil {
		t.Fatalf("Should have gotten an error: %v", err)
	}

	if !strings.HasPrefix(err.Error(), "HTTP status 404") {
		t.Fatalf("Should start with 'HTTP status' instead of: %v", err.Error())
	}

}

func TestWrongFormatResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This is text."))
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	_, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	if pbReq.Bidders[0].Debug[0].StatusCode != 200 {
		t.Fatalf("StatusCode should be 200 instead of: %v", pbReq.Bidders[0].Debug[0].StatusCode)
	}

	if err == nil {
		t.Fatalf("Should have gotten an error: %v", err)
	}

	if !strings.HasPrefix(err.Error(), "invalid character") {
		t.Fatalf("Should start with 'invalid character' instead of: %v", err)
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
	bids, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	if len(bids) != 0 {
		t.Fatalf("Length of bids should be 0 instead of: %v", len(bids))
	}

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
	bids, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	if len(bids) != 0 {
		t.Fatalf("Length of bids should be 0 instead of: %v", len(bids))
	}

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
			Ext:   json.RawMessage("{\"rp\":{\"targeting\":[{\"key\":\"key1\",\"values\":[\"value1\"]},{\"key\":\"key2\",\"values\":[\"value2\"]}]}}"),
		}
		js, _ := json.Marshal(resp)
		w.Write(js)
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	bids, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	if len(bids) != 0 {
		t.Fatalf("Length of bids should be 0 instead of: %v", len(bids))
	}

	if err == nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}

	if !strings.HasPrefix(err.Error(), "Unknown ad unit code") {
		t.Fatalf("Should start with 'Unknown ad unit code' instead of: %v", err)
	}

}

func TestZeroPriceBidResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openrtb.BidResponse{
			ID:    "test-response-id",
			BidID: "test-bid-id",
			Cur:   "USD",
			SeatBid: []openrtb.SeatBid{
				{
					Seat: "RUBICON",
					Bid:  make([]openrtb.Bid, 1),
				},
			},
		}
		resp.SeatBid[0].Bid[0] = openrtb.Bid{
			ID:    "test-bid-id",
			ImpID: "first-tag",
			Price: 0,
			AdM:   "zma",
			Ext:   json.RawMessage("{\"rp\":{\"targeting\":[{\"key\":\"key1\",\"values\":[\"value1\"]},{\"key\":\"key2\",\"values\":[\"value2\"]}]}}"),
		}
		js, _ := json.Marshal(resp)
		w.Write(js)
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	b, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	if b != nil {
		t.Fatalf("\n\n\n0 price bids are being included %d, err : %v", len(b), err)
	}

}

func TestDifferentRequest(t *testing.T) {
	SIZE_ID := getTestSizes()
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
	_, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err == nil {
		t.Fatalf("Should have gotten an error: %v", err)
	}

	// set media back to normal
	pbReq.Bidders[0].AdUnits[0].MediaTypes = []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}

	// test wrong params
	pbReq.Bidders[0].AdUnits[0].Params = json.RawMessage(fmt.Sprintf("{\"zoneId\": %s, \"siteId\": %d, \"visitor\": %s, \"inventory\": %s}", "zma", rubidata.siteID, rubidata.visitorTargeting, rubidata.inventoryTargeting))
	_, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err == nil {
		t.Fatalf("Should have gotten an error: %v", err)
	}

	// set params back to normal
	pbReq.Bidders[0].AdUnits[0].Params = json.RawMessage(fmt.Sprintf("{\"zoneId\": %d, \"siteId\": %d, \"accountId\": %d, \"visitor\": %s, \"inventory\": %s}", 8394, rubidata.siteID, rubidata.accountID, rubidata.visitorTargeting, rubidata.inventoryTargeting))

	// test invalid size
	pbReq.Bidders[0].AdUnits[0].Sizes = []openrtb.Format{
		{
			W: 2222,
			H: 333,
		},
	}
	pbReq.Bidders[0].AdUnits[1].Sizes = []openrtb.Format{
		{
			W: 222,
			H: 3333,
		},
		{
			W: 350,
			H: 270,
		},
	}
	pbReq.Bidders[0].AdUnits = pbReq.Bidders[0].AdUnits[:len(pbReq.Bidders[0].AdUnits)-1]
	b, err := an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err == nil || len(b) != 0 {
		t.Fatalf("Filtering bids based on ad unit sizes failed. Got %d bids instead of 0", len(b))
	}

	pbReq.Bidders[0].AdUnits[1].Sizes = []openrtb.Format{
		{
			W: 222,
			H: 3333,
		},
		SIZE_ID[10],
		SIZE_ID[15],
	}
	b, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err != nil || len(b) != 1 {
		t.Fatalf("Filtering bids based on ad unit sizes failed. Got %d bids instead of 1, error = %v", len(b), err)
	}
}

func CreatePrebidRequest(server *httptest.Server, t *testing.T) (an *RubiconAdapter, ctx context.Context, pbReq *pbs.PBSRequest) {
	SIZE_ID := getTestSizes()
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
	an = NewRubiconAdapter(&conf, "uri", rubidata.xapiuser, rubidata.xapipass, "pbs-test-tracker")
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
				SIZE_ID[10],
				SIZE_ID[15],
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

	req := httptest.NewRequest("POST", server.URL, body)
	req.Header.Add("Referer", rubidata.page)
	req.Header.Add("User-Agent", rubidata.deviceUA)
	req.Header.Add("X-Real-IP", rubidata.deviceIP)

	pc := usersync.ParsePBSCookieFromRequest(req, &config.HostCookie{})
	pc.TrySync("rubicon", rubidata.buyerUID)
	fakewriter := httptest.NewRecorder()
	pc.SetCookieOnResponse(fakewriter, "", 90*24*time.Hour)
	req.Header.Add("Cookie", fakewriter.Header().Get("Set-Cookie"))

	cacheClient, _ := dummycache.New()
	hcc := config.HostCookie{}

	pbReq, err = pbs.ParsePBSRequest(req, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, cacheClient, &hcc)
	pbReq.IsDebug = true
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

func TestOpenRTBRequest(t *testing.T) {
	SIZE_ID := getTestSizes()
	bidder := new(RubiconAdapter)

	rubidata = rubiBidInfo{
		domain:        "nytimes.com",
		page:          "https://www.nytimes.com/2017/05/04/movies/guardians-of-the-galaxy-2-review-chris-pratt.html?hpw&rref=movies&action=click&pgtype=Homepage&module=well-region&region=bottom-well&WT.nav=bottom-well&_r=0",
		deviceIP:      "25.91.96.36",
		deviceUA:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_4) AppleWebKit/603.1.30 (KHTML, like Gecko) Version/10.1 Safari/603.1.30",
		buyerUID:      "need-an-actual-rp-id",
		devicePxRatio: 4.0,
	}

	request := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{{
			ID: "test-imp-banner-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{
					SIZE_ID[15],
					SIZE_ID[10],
				},
			},
			Ext: json.RawMessage(`{"bidder": {
				"zoneId": 8394,
				"siteId": 283282,
				"accountId": 7891,
				"inventory": {"key1" : "val1"},
				"visitor": {"key2" : "val2"}
			}}`),
		}, {
			ID: "test-imp-video-id",
			Video: &openrtb.Video{
				W:           640,
				H:           360,
				MIMEs:       []string{"video/mp4"},
				MinDuration: 15,
				MaxDuration: 30,
			},
			Ext: json.RawMessage(`{"bidder": {
				"zoneId": 7780,
				"siteId": 283282,
				"accountId": 7891,
				"inventory": {"key1" : "val1"},
				"visitor": {"key2" : "val2"},
				"video": {
					"language": "en",
					"playerHeight": 360,
					"playerWidth": 640,
					"size_id": 203,
					"skip": 1,
					"skipdelay": 5
				}
			}}`),
		}},
		Device: &openrtb.Device{
			PxRatio: rubidata.devicePxRatio,
		},
		User: &openrtb.User{
			Ext: json.RawMessage(`{"digitrust": {
                    "id": "some-digitrust-id",
                    "keyv": 1,
                    "pref": 0
                },
                "tpid": [{
                    "source": "tdid",
                    "uid": "3d50a262-bd8e-4be3-90b8-246291523907"
                },{
                    "source": "pubcid",
                    "uid": "2402fc76-7b39-4f0e-bfc2-060ef7693648"
                }]
            }`),
		},
	}

	reqs, errs := bidder.MakeRequests(request)

	if len(errs) > 0 {
		t.Errorf("Got unexpected errors while building HTTP requests: %v", errs)
	}
	if len(reqs) != 2 {
		t.Fatalf("Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 2)
	}

	for i := 0; i < len(reqs); i++ {
		httpReq := reqs[i]
		if httpReq.Method != "POST" {
			t.Errorf("Expected a POST message. Got %s", httpReq.Method)
		}

		var rpRequest openrtb.BidRequest
		if err := json.Unmarshal(httpReq.Body, &rpRequest); err != nil {
			t.Fatalf("Failed to unmarshal HTTP request: %v", rpRequest)
		}

		if rpRequest.ID != request.ID {
			t.Errorf("Bad Request ID. Expected %s, Got %s", request.ID, rpRequest.ID)
		}
		if len(rpRequest.Imp) != len(request.Imp) {
			t.Fatalf("Wrong len(request.Imp). Expected %d, Got %d", len(request.Imp), len(rpRequest.Imp))
		}
		if rpRequest.Cur != nil {
			t.Fatalf("Wrong request.Cur. Expected nil, Got %s", rpRequest.Cur)
		}

		if rpRequest.Imp[0].ID == "test-imp-banner-id" {
			var rpExt rubiconBannerExt
			if err := json.Unmarshal(rpRequest.Imp[0].Ext, &rpExt); err != nil {
				t.Fatal("Error unmarshalling request from the outgoing request.")
			}

			if rpRequest.Imp[0].Banner.Format[0].W != 300 {
				t.Fatalf("Banner width does not match. Expected %d, Got %d", 300, rpRequest.Imp[0].Banner.Format[0].W)
			}
			if rpRequest.Imp[0].Banner.Format[0].H != 250 {
				t.Fatalf("Banner height does not match. Expected %d, Got %d", 250, rpRequest.Imp[0].Banner.Format[0].H)
			}
			if rpRequest.Imp[0].Banner.Format[1].W != 300 {
				t.Fatalf("Banner width does not match. Expected %d, Got %d", 300, rpRequest.Imp[0].Banner.Format[1].W)
			}
			if rpRequest.Imp[0].Banner.Format[1].H != 600 {
				t.Fatalf("Banner height does not match. Expected %d, Got %d", 600, rpRequest.Imp[0].Banner.Format[1].H)
			}
		} else if rpRequest.Imp[0].ID == "test-imp-video-id" {
			var rpExt rubiconVideoExt
			if err := json.Unmarshal(rpRequest.Imp[0].Ext, &rpExt); err != nil {
				t.Fatal("Error unmarshalling request from the outgoing request.")
			}

			if rpRequest.Imp[0].Video.W != 640 {
				t.Fatalf("Video width does not match. Expected %d, Got %d", 640, rpRequest.Imp[0].Video.W)
			}
			if rpRequest.Imp[0].Video.H != 360 {
				t.Fatalf("Video height does not match. Expected %d, Got %d", 360, rpRequest.Imp[0].Video.H)
			}
			if rpRequest.Imp[0].Video.MIMEs[0] != "video/mp4" {
				t.Fatalf("Video MIMEs do not match. Expected %s, Got %s", "video/mp4", rpRequest.Imp[0].Video.MIMEs[0])
			}
			if rpRequest.Imp[0].Video.MinDuration != 15 {
				t.Fatalf("Video min duration does not match. Expected %d, Got %d", 15, rpRequest.Imp[0].Video.MinDuration)
			}
			if rpRequest.Imp[0].Video.MaxDuration != 30 {
				t.Fatalf("Video max duration does not match. Expected %d, Got %d", 30, rpRequest.Imp[0].Video.MaxDuration)
			}
		}

		if rpRequest.User.Ext != nil {
			var userExt rubiconUserExt
			if err := json.Unmarshal(rpRequest.User.Ext, &userExt); err != nil {
				t.Fatal("Error unmarshalling request.user.ext object.")
			}
			if userExt.DigiTrust.ID != "some-digitrust-id" || userExt.DigiTrust.KeyV != 1 || userExt.DigiTrust.Pref != 0 {
				t.Fatal("DigiTrust values are not as expected!")
			}
			if userExt.TpID == nil || len(userExt.TpID) != 2 {
				t.Fatal("TpID values are not as expected!")
			}
		} else {
			t.Fatalf("User.Ext object should not be nil since it contains a valid digitrust object.")
		}
	}
}

func TestOpenRTBRequestWithBannerImpEvenIfImpHasVideo(t *testing.T) {
	SIZE_ID := getTestSizes()
	bidder := new(RubiconAdapter)

	request := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{
					SIZE_ID[15],
					SIZE_ID[10],
				},
			},
			Video: &openrtb.Video{
				W:     640,
				H:     360,
				MIMEs: []string{"video/mp4"},
			},
			Ext: json.RawMessage(`{"bidder": {
				"zoneId": 8394,
				"siteId": 283282,
				"accountId": 7891,
				"inventory": {"key1" : "val1"},
				"visitor": {"key2" : "val2"}
			}}`),
		}},
	}

	reqs, errs := bidder.MakeRequests(request)

	if len(errs) > 0 {
		t.Errorf("Got unexpected errors while building HTTP requests: %v", errs)
	}
	if len(reqs) != 1 {
		t.Fatalf("Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 1)
	}

	rubiconReq := &openrtb.BidRequest{}
	if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
		t.Errorf("Unexpected error while decoding request: %s", err)
	}

	if len(rubiconReq.Imp) != 1 {
		t.Fatalf("Unexpected number of request impressions. Got %d. Expected %d", len(rubiconReq.Imp), 1)
	}

	if rubiconReq.Imp[0].Video != nil {
		t.Error("Unexpected video object in request impression")
	}

	if rubiconReq.Imp[0].Banner == nil {
		t.Error("Banner object must be in request impression")
	}
}

func TestOpenRTBRequestWithVideoImpEvenIfImpHasBannerButAllRequiredVideoFields(t *testing.T) {
	SIZE_ID := getTestSizes()
	bidder := new(RubiconAdapter)

	request := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{
					SIZE_ID[15],
					SIZE_ID[10],
				},
			},
			Video: &openrtb.Video{
				W:           640,
				H:           360,
				MIMEs:       []string{"video/mp4"},
				Protocols:   []openrtb.Protocol{openrtb.ProtocolVAST10},
				MaxDuration: 30,
				Linearity:   1,
				API:         []openrtb.APIFramework{},
			},
			Ext: json.RawMessage(`{"bidder": {
				"zoneId": 8394,
				"siteId": 283282,
				"accountId": 7891,
				"inventory": {"key1" : "val1"},
				"visitor": {"key2" : "val2"}
			}}`),
		}},
	}

	reqs, errs := bidder.MakeRequests(request)

	if len(errs) > 0 {
		t.Errorf("Got unexpected errors while building HTTP requests: %v", errs)
	}
	if len(reqs) != 1 {
		t.Fatalf("Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 1)
	}

	rubiconReq := &openrtb.BidRequest{}
	if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
		t.Errorf("Unexpected error while decoding request: %s", err)
	}

	if len(rubiconReq.Imp) != 1 {
		t.Fatalf("Unexpected number of request impressions. Got %d. Expected %d", len(rubiconReq.Imp), 1)
	}

	if rubiconReq.Imp[0].Banner != nil {
		t.Error("Unexpected banner object in request impression")
	}

	if rubiconReq.Imp[0].Video == nil {
		t.Error("Video object must be in request impression")
	}
}

func TestOpenRTBEmptyResponse(t *testing.T) {
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusNoContent,
	}
	bidder := new(RubiconAdapter)
	bidResponse, errs := bidder.MakeBids(nil, nil, httpResp)
	if bidResponse != nil && len(bidResponse.Bids) != 0 {
		t.Errorf("Expected 0 bids. Got %d", len(bidResponse.Bids))
	}
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors. Got %d", len(errs))
	}
}

func TestOpenRTBSurpriseResponse(t *testing.T) {
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusAccepted,
	}
	bidder := new(RubiconAdapter)
	bidResponse, errs := bidder.MakeBids(nil, nil, httpResp)
	if bidResponse != nil && len(bidResponse.Bids) != 0 {
		t.Errorf("Expected 0 bids. Got %d", len(bidResponse.Bids))
	}
	if len(errs) != 1 {
		t.Errorf("Expected 1 error. Got %d", len(errs))
	}
}

func TestOpenRTBStandardResponse(t *testing.T) {
	request := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 320,
					H: 50,
				}},
			},
			Ext: json.RawMessage(`{"bidder": {
				"accountId": 2763,
				"siteId": 68780,
				"zoneId": 327642
			}}`),
		}},
	}

	requestJson, _ := json.Marshal(request)
	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     "test-uri",
		Body:    requestJson,
		Headers: nil,
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"test-request-id","seatbid":[{"bid":[{"id":"1234567890","impid":"test-imp-id","price": 2,"crid":"4122982","adm":"some ad","h": 50,"w": 320,"ext":{"bidder":{"rp":{"targeting": {"key": "rpfl_2763", "values":["43_tier0100"]},"mime": "text/html","size_id": 43}}}}]}]}`),
	}

	bidder := new(RubiconAdapter)
	bidResponse, errs := bidder.MakeBids(request, reqData, httpResp)

	if bidResponse != nil && len(bidResponse.Bids) != 1 {
		t.Fatalf("Expected 1 bid. Got %d", len(bidResponse.Bids))
	}
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors. Got %d", len(errs))
	}
	if bidResponse.Bids[0].BidType != openrtb_ext.BidTypeBanner {
		t.Errorf("Expected a banner bid. Got: %s", bidResponse.Bids[0].BidType)
	}
	theBid := bidResponse.Bids[0].Bid
	if theBid.ID != "1234567890" {
		t.Errorf("Bad bid ID. Expected %s, got %s", "1234567890", theBid.ID)
	}
}

func TestOpenRTBCopyBidIdFromResponseIfZero(t *testing.T) {
	request := &openrtb.BidRequest{
		ID:  "test-request-id",
		Imp: []openrtb.Imp{{}},
	}

	requestJson, _ := json.Marshal(request)
	reqData := &adapters.RequestData{Body: requestJson}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"test-request-id","bidid":"1234567890","seatbid":[{"bid":[{"id":"0","price": 1}]}]}`),
	}

	bidder := new(RubiconAdapter)
	bidResponse, _ := bidder.MakeBids(request, reqData, httpResp)

	theBid := bidResponse.Bids[0].Bid
	if theBid.ID != "1234567890" {
		t.Errorf("Bad bid ID. Expected %s, got %s", "1234567890", theBid.ID)
	}
}

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "rubicontest", NewRubiconBidder(http.DefaultClient, "uri", "xuser", "xpass", "pbs-test-tracker"))
}
