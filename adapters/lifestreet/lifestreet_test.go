package lifestreet

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/usersync"

	"fmt"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
)

type lsTagInfo struct {
	code    string
	slotTag string
	bid     float64
	content string
}

type lsBidInfo struct {
	appBundle            string
	deviceIP             string
	deviceUA             string
	deviceMake           string
	deviceModel          string
	deviceConnectiontype int8
	deviceIfa            string
	tags                 []lsTagInfo
	referrer             string
	width                uint64
	height               uint64
	delay                time.Duration
}

var lsdata lsBidInfo

func DummyLifestreetServer(w http.ResponseWriter, r *http.Request) {
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

	if breq.App == nil {
		http.Error(w, fmt.Sprintf("No app object sent"), http.StatusInternalServerError)
		return
	}
	if breq.App.Bundle != lsdata.appBundle {
		http.Error(w, fmt.Sprintf("Bundle '%s' doesn't match '%s", breq.App.Bundle, lsdata.appBundle), http.StatusInternalServerError)
		return
	}
	if breq.Device.UA != lsdata.deviceUA {
		http.Error(w, fmt.Sprintf("UA '%s' doesn't match '%s", breq.Device.UA, lsdata.deviceUA), http.StatusInternalServerError)
		return
	}
	if breq.Device.IP != lsdata.deviceIP {
		http.Error(w, fmt.Sprintf("IP '%s' doesn't match '%s", breq.Device.IP, lsdata.deviceIP), http.StatusInternalServerError)
		return
	}
	if breq.Device.Make != lsdata.deviceMake {
		http.Error(w, fmt.Sprintf("Make '%s' doesn't match '%s", breq.Device.Make, lsdata.deviceMake), http.StatusInternalServerError)
		return
	}
	if breq.Device.Model != lsdata.deviceModel {
		http.Error(w, fmt.Sprintf("Model '%s' doesn't match '%s", breq.Device.Model, lsdata.deviceModel), http.StatusInternalServerError)
		return
	}
	if *breq.Device.ConnectionType != openrtb.ConnectionType(lsdata.deviceConnectiontype) {
		http.Error(w, fmt.Sprintf("Connectiontype '%d' doesn't match '%d", breq.Device.ConnectionType, lsdata.deviceConnectiontype), http.StatusInternalServerError)
		return
	}
	if breq.Device.IFA != lsdata.deviceIfa {
		http.Error(w, fmt.Sprintf("IFA '%s' doesn't match '%s", breq.Device.IFA, lsdata.deviceIfa), http.StatusInternalServerError)
		return
	}
	if len(breq.Imp) != 1 {
		http.Error(w, fmt.Sprintf("Wrong number of imp objects sent: %d", len(breq.Imp)), http.StatusInternalServerError)
		return
	}
	var bid *openrtb.Bid
	for _, tag := range lsdata.tags {
		if breq.Imp[0].Banner == nil {
			http.Error(w, fmt.Sprintf("No banner object sent"), http.StatusInternalServerError)
			return
		}
		if *breq.Imp[0].Banner.W != lsdata.width || *breq.Imp[0].Banner.H != lsdata.height {
			http.Error(w, fmt.Sprintf("Size '%dx%d' doesn't match '%dx%d", breq.Imp[0].Banner.W, breq.Imp[0].Banner.H, lsdata.width, lsdata.height), http.StatusInternalServerError)
			return
		}
		if breq.Imp[0].TagID == tag.slotTag {
			bid = &openrtb.Bid{
				ID:    "random-id",
				ImpID: breq.Imp[0].ID,
				Price: tag.bid,
				AdM:   tag.content,
				W:     lsdata.width,
				H:     lsdata.height,
			}
		}
	}
	if bid == nil {
		http.Error(w, fmt.Sprintf("Slot tag '%s' not found", breq.Imp[0].TagID), http.StatusInternalServerError)
		return
	}

	resp := openrtb.BidResponse{
		ID:    "2345676337",
		BidID: "975537589956",
		Cur:   "USD",
		SeatBid: []openrtb.SeatBid{
			{
				Seat: "LSM",
				Bid:  []openrtb.Bid{*bid},
			},
		},
	}

	if lsdata.delay > 0 {
		<-time.After(lsdata.delay)
	}

	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func TestLifestreetBasicResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyLifestreetServer))
	defer server.Close()

	lsdata = lsBidInfo{
		appBundle:            "AppNexus.PrebidMobileDemo",
		deviceIP:             "111.111.111.111",
		deviceUA:             "Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_1 like Mac OS X) AppleWebKit/603.1.30 (KHTML, like Gecko) Mobile/14E8301",
		deviceMake:           "Apple",
		deviceModel:          "x86_64",
		deviceConnectiontype: 1,
		deviceIfa:            "6F3EA622-C2EE-4449-A97A-AE986D080C08",
		tags:                 make([]lsTagInfo, 2),
		referrer:             "http://test.com",
		width:                320,
		height:               480,
	}
	lsdata.tags[0] = lsTagInfo{
		code:    "first-tag",
		slotTag: "slot123.123",
		bid:     2.44,
	}
	lsdata.tags[1] = lsTagInfo{
		code:    "second-tag",
		slotTag: "slot122.122",
		bid:     1.11,
	}

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewLifestreetAdapter(&conf, "https://prebid.s2s.lfstmedia.com/adrequest")
	an.URI = server.URL

	pbin := pbs.PBSRequest{
		AdUnits: make([]pbs.AdUnit, 2),
		App: &openrtb.App{
			Bundle: lsdata.appBundle,
		},
		Device: &openrtb.Device{
			UA:             lsdata.deviceUA,
			IP:             lsdata.deviceIP,
			Make:           lsdata.deviceMake,
			Model:          lsdata.deviceModel,
			ConnectionType: openrtb.ConnectionType(lsdata.deviceConnectiontype).Ptr(),
			IFA:            lsdata.deviceIfa,
		},
	}
	for i, tag := range lsdata.tags {
		pbin.AdUnits[i] = pbs.AdUnit{
			Code: tag.code,
			Sizes: []openrtb.Format{
				{
					W: lsdata.width,
					H: lsdata.height,
				},
			},
			Bids: []pbs.Bids{
				{
					BidderCode: "lifestreet",
					BidID:      fmt.Sprintf("random-id-from-pbjs-%d", i),
					Params:     json.RawMessage(fmt.Sprintf("{\"slot_tag\": \"%s\"}", tag.slotTag)),
				},
			},
		}
	}

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(pbin)
	if err != nil {
		t.Fatalf("Json encoding failed: %v", err)
	}

	fmt.Println("body", body)

	req := httptest.NewRequest("POST", server.URL, body)
	req.Header.Add("User-Agent", lsdata.deviceUA)
	req.Header.Add("Referer", lsdata.referrer)
	req.Header.Add("X-Real-IP", lsdata.deviceIP)

	pc := usersync.ParsePBSCookieFromRequest(req, &config.HostCookie{})
	fakewriter := httptest.NewRecorder()

	pc.SetCookieOnResponse(fakewriter, false, &config.HostCookie{Domain: ""}, 90*24*time.Hour)
	req.Header.Add("Cookie", fakewriter.Header().Get("Set-Cookie"))

	cacheClient, _ := dummycache.New()
	hcc := config.HostCookie{}
	pbReq, err := pbs.ParsePBSRequest(req, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, cacheClient, &hcc)
	if err != nil {
		t.Fatalf("ParsePBSRequest failed: %v", err)
	}
	if len(pbReq.Bidders) != 1 {
		t.Fatalf("ParsePBSRequest returned %d bidders instead of 1", len(pbReq.Bidders))
	}
	if pbReq.Bidders[0].BidderCode != "lifestreet" {
		t.Fatalf("ParsePBSRequest returned invalid bidder")
	}

	ctx := context.TODO()
	bids, err := an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}
	if len(bids) != 2 {
		t.Fatalf("Received %d bids instead of 2", len(bids))
	}
	for _, bid := range bids {
		matched := false
		for _, tag := range lsdata.tags {
			if bid.AdUnitCode == tag.code {
				matched = true
				if bid.BidderCode != "lifestreet" {
					t.Errorf("Incorrect BidderCode '%s'", bid.BidderCode)
				}
				if bid.Price != tag.bid {
					t.Errorf("Incorrect bid price '%.2f' expected '%.2f'", bid.Price, tag.bid)
				}
				if bid.Width != lsdata.width || bid.Height != lsdata.height {
					t.Errorf("Incorrect bid size %dx%d, expected %dx%d", bid.Width, bid.Height, lsdata.width, lsdata.height)
				}
				if bid.Adm != tag.content {
					t.Errorf("Incorrect bid markup '%s' expected '%s'", bid.Adm, tag.content)
				}
			}
		}
		if !matched {
			t.Errorf("Received bid for unknown ad unit '%s'", bid.AdUnitCode)
		}
	}

	// same test but with request timing out
	lsdata.delay = 5 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	bids, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err == nil {
		t.Fatalf("Should have gotten a timeout error: %v", err)
	}
}
