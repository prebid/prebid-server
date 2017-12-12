package admixer

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

	"fmt"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
)

type admixerTagInfo struct {
	code    string
	zoneOId string
	bid     float64
	content string
}

type admixerBidInfo struct {
	appBundle            string
	deviceIP             string
	deviceUA             string
	deviceMake           string
	deviceModel          string
	deviceConnectiontype int8
	deviceIfa            string
	tags                 []admixerTagInfo
	referrer             string
	width                uint64
	height               uint64
	delay                time.Duration
}

var admixerData admixerBidInfo

func DummyAdmixerServer(w http.ResponseWriter, r *http.Request) {
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
	if breq.App.Bundle != admixerData.appBundle {
		http.Error(w, fmt.Sprintf("Bundle '%s' doesn't match '%s", breq.App.Bundle, admixerData.appBundle), http.StatusInternalServerError)
		return
	}
	if breq.Device.UA != admixerData.deviceUA {
		http.Error(w, fmt.Sprintf("UA '%s' doesn't match '%s", breq.Device.UA, admixerData.deviceUA), http.StatusInternalServerError)
		return
	}
	if breq.Device.IP != admixerData.deviceIP {
		http.Error(w, fmt.Sprintf("IP '%s' doesn't match '%s", breq.Device.IP, admixerData.deviceIP), http.StatusInternalServerError)
		return
	}
	if breq.Device.Make != admixerData.deviceMake {
		http.Error(w, fmt.Sprintf("Make '%s' doesn't match '%s", breq.Device.Make, admixerData.deviceMake), http.StatusInternalServerError)
		return
	}
	if breq.Device.Model != admixerData.deviceModel {
		http.Error(w, fmt.Sprintf("Model '%s' doesn't match '%s", breq.Device.Model, admixerData.deviceModel), http.StatusInternalServerError)
		return
	}
	if *breq.Device.ConnectionType != openrtb.ConnectionType(admixerData.deviceConnectiontype) {
		http.Error(w, fmt.Sprintf("Connectiontype '%d' doesn't match '%d", breq.Device.ConnectionType, admixerData.deviceConnectiontype), http.StatusInternalServerError)
		return
	}
	if breq.Device.IFA != admixerData.deviceIfa {
		http.Error(w, fmt.Sprintf("IFA '%s' doesn't match '%s", breq.Device.IFA, admixerData.deviceIfa), http.StatusInternalServerError)
		return
	}
	if len(breq.Imp) != 1 {
		http.Error(w, fmt.Sprintf("Wrong number of imp objects sent: %d", len(breq.Imp)), http.StatusInternalServerError)
		return
	}
	var bid *openrtb.Bid
	for _, tag := range admixerData.tags {
		if breq.Imp[0].Banner == nil {
			http.Error(w, fmt.Sprintf("No banner object sent"), http.StatusInternalServerError)
			return
		}
		if *breq.Imp[0].Banner.W != admixerData.width || *breq.Imp[0].Banner.H != admixerData.height {
			http.Error(w, fmt.Sprintf("Size '%dx%d' doesn't match '%dx%d", breq.Imp[0].Banner.W, breq.Imp[0].Banner.H, admixerData.width, admixerData.height), http.StatusInternalServerError)
			return
		}
		if breq.Imp[0].TagID == tag.zoneOId {
			bid = &openrtb.Bid{
				ID:    "1",
				ImpID: breq.Imp[0].ID,
				Price: tag.bid,
				AdM:   tag.content,
				W:     admixerData.width,
				H:     admixerData.height,
			}
		}
	}
	if bid == nil {
		http.Error(w, fmt.Sprintf("Slot tag '%s' not found", breq.Imp[0].TagID), http.StatusInternalServerError)
		return
	}

	resp := openrtb.BidResponse{
		ID:    "1",
		BidID: "3456346",
		Cur:   "USD",
		SeatBid: []openrtb.SeatBid{
			{
				Seat: "Seat",
				Bid:  []openrtb.Bid{*bid},
			},
		},
	}

	if admixerData.delay > 0 {
		<-time.After(admixerData.delay)
	}

	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func TestAdmixerBasicResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyAdmixerServer))
	defer server.Close()

	admixerData = admixerBidInfo{
		appBundle:            "AppNexus.PrebidMobileDemo",
		deviceIP:             "111.111.111.111",
		deviceUA:             "Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_1 like Mac OS X) AppleWebKit/603.1.30 (KHTML, like Gecko) Mobile/14E8301",
		deviceMake:           "Apple",
		deviceModel:          "x86_64",
		deviceConnectiontype: 1,
		deviceIfa:            "6F3EA622-C2EE-4449-A97A-AE986D080C08",
		tags:                 make([]admixerTagInfo, 2),
		referrer:             "http://test.com",
		width:                320,
		height:               480,
	}
	admixerData.tags[0] = admixerTagInfo{
		code:    "first-tag",
		zoneOId: "B750AD52-6A57-43C9-B180-3311409CF2C1",
		bid:     2.44,
	}
	admixerData.tags[1] = admixerTagInfo{
		code:    "second-tag",
		zoneOId: "69A25E8C-EF50-4CE7-956E-32C6D4133AA8",
		bid:     1.11,
	}

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewAdmixerAdapter(&conf, server.URL)
	an.URI = server.URL

	pbin := pbs.PBSRequest{
		AdUnits: make([]pbs.AdUnit, 2),
		App: &openrtb.App{
			Bundle: admixerData.appBundle,
		},
		Device: &openrtb.Device{
			UA:             admixerData.deviceUA,
			IP:             admixerData.deviceIP,
			Make:           admixerData.deviceMake,
			Model:          admixerData.deviceModel,
			ConnectionType: openrtb.ConnectionType(admixerData.deviceConnectiontype).Ptr(),
			IFA:            admixerData.deviceIfa,
		},
	}
	for i, tag := range admixerData.tags {
		pbin.AdUnits[i] = pbs.AdUnit{
			Code: tag.code,
			Sizes: []openrtb.Format{
				{
					W: admixerData.width,
					H: admixerData.height,
				},
			},
			Bids: []pbs.Bids{
				pbs.Bids{
					BidderCode: "admixer",
					BidID:      fmt.Sprintf("random-id-from-pbjs-%d", i),
					Params:     json.RawMessage(fmt.Sprintf("{\"zoneOId\": \"%s\"}", tag.zoneOId)),
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
	req.Header.Add("User-Agent", admixerData.deviceUA)
	req.Header.Add("Referer", admixerData.referrer)
	req.Header.Add("X-Real-IP", admixerData.deviceIP)

	pc := pbs.ParsePBSCookieFromRequest(req, &config.Cookie{})
	fakewriter := httptest.NewRecorder()
	pc.SetCookieOnResponse(fakewriter, "")
	req.Header.Add("Cookie", fakewriter.Header().Get("Set-Cookie"))

	cacheClient, _ := dummycache.New()
	hcs := pbs.HostCookieSettings{}
	pbReq, err := pbs.ParsePBSRequest(req, cacheClient, &hcs)
	if err != nil {
		t.Fatalf("ParsePBSRequest failed: %v", err)
	}
	if len(pbReq.Bidders) != 1 {
		t.Fatalf("ParsePBSRequest returned %d bidders instead of 1", len(pbReq.Bidders))
	}
	if pbReq.Bidders[0].BidderCode != "admixer" {
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
		for _, tag := range admixerData.tags {
			if bid.AdUnitCode == tag.code {
				matched = true
				if bid.BidderCode != "admixer" {
					t.Errorf("Incorrect BidderCode '%s'", bid.BidderCode)
				}
				if bid.Price != tag.bid {
					t.Errorf("Incorrect bid price '%.2f' expected '%.2f'", bid.Price, tag.bid)
				}
				if bid.Width != admixerData.width || bid.Height != admixerData.height {
					t.Errorf("Incorrect bid size %dx%d, expected %dx%d", bid.Width, bid.Height, admixerData.width, admixerData.height)
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
	admixerData.delay = 5 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	bids, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err == nil {
		t.Fatalf("Should have gotten a timeout error: %v", err)
	}
}

func TestAdmixerUserSyncInfo(t *testing.T) {
	url := "//inv-nets.admixer.net/cm.aspx?ssp=prebid&rurl=localhost%2Fsetuid%3Fbidder%3Dam-uid%26uid%3D%24%24visitor_cookie%24%24"

	an := NewAdmixerAdapter(adapters.DefaultHTTPAdapterConfig, "localhost")
	if an.usersyncInfo.URL != url {
		t.Fatalf("User Sync Info URL '%s' doesn't match '%s'", an.usersyncInfo.URL, url)
	}
	if an.usersyncInfo.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if an.usersyncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
