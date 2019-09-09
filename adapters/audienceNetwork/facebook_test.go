package audienceNetwork

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PubMatic-OpenWrap/prebid-server/cache/dummycache"
	"github.com/PubMatic-OpenWrap/prebid-server/pbs"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"

	"fmt"

	"github.com/mxmCherry/openrtb"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/config"
)

type tagInfo struct {
	code        string
	placementID string
	bid         float64
	content     string
	delay       time.Duration
	W           uint64
	H           uint64
	Instl       int8
}

type bidInfo struct {
	partnerID   int
	domain      string
	page        string
	publisherID string
	tags        []tagInfo
	deviceIP    string
	deviceUA    string
	buyerUID    string
}

var fbdata bidInfo

type FacebookExt struct {
	PlatformID int `json:"platformid"`
}

func DummyFacebookServer(w http.ResponseWriter, r *http.Request) {
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

	if string(breq.Ext) == "" {
		http.Error(w, "No Ext field provided", http.StatusInternalServerError)
		return
	}
	var fext FacebookExt
	err = json.Unmarshal(breq.Ext, &fext)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if fext.PlatformID != fbdata.partnerID {
		http.Error(w, fmt.Sprintf("Platform ID '%d' doesn't match '%d", fext.PlatformID, fbdata.partnerID), http.StatusInternalServerError)
		return
	}
	if breq.Site == nil {
		http.Error(w, fmt.Sprintf("No site object sent"), http.StatusInternalServerError)
		return
	}
	if breq.Site.Domain != fbdata.domain {
		http.Error(w, fmt.Sprintf("Domain '%s' doesn't match '%s", breq.Site.Domain, fbdata.domain), http.StatusInternalServerError)
		return
	}
	if breq.Site.Page != fbdata.page {
		http.Error(w, fmt.Sprintf("Page '%s' doesn't match '%s", breq.Site.Page, fbdata.page), http.StatusInternalServerError)
		return
	}
	if breq.Device.UA != fbdata.deviceUA {
		http.Error(w, fmt.Sprintf("UA '%s' doesn't match '%s", breq.Device.UA, fbdata.deviceUA), http.StatusInternalServerError)
		return
	}
	if breq.Device.IP != fbdata.deviceIP {
		http.Error(w, fmt.Sprintf("IP '%s' doesn't match '%s", breq.Device.IP, fbdata.deviceIP), http.StatusInternalServerError)
		return
	}
	if breq.User.BuyerUID != fbdata.buyerUID {
		http.Error(w, fmt.Sprintf("User ID '%s' doesn't match '%s", breq.User.BuyerUID, fbdata.buyerUID), http.StatusInternalServerError)
		return
	}
	if len(breq.Imp) != 1 {
		http.Error(w, fmt.Sprintf("Wrong number of imp objects sent: %d", len(breq.Imp)), http.StatusInternalServerError)
		return
	}
	var bid *openrtb.Bid
	for _, tag := range fbdata.tags {
		if breq.Imp[0].Banner == nil {
			http.Error(w, fmt.Sprintf("No banner object sent"), http.StatusInternalServerError)
			return
		}
		if breq.Imp[0].Instl == 0 {
			supportedHeight := map[uint64]bool{
				50:  true,
				90:  true,
				250: true,
			}
			if !supportedHeight[*breq.Imp[0].Banner.H] {
				http.Error(w, fmt.Sprintf("Height '%d' not supported", breq.Imp[0].Banner.H), http.StatusBadRequest)
				return
			}
		} else if breq.Imp[0].Instl == 1 {
			if *breq.Imp[0].Banner.H != 0 || *breq.Imp[0].Banner.W != 0 {
				http.Error(w, fmt.Sprintf("Width and height should be 0, 0 for instl type"), http.StatusBadRequest)
				return
			}
		} else {
			http.Error(w, fmt.Sprintf("Invalid Instl sent"), http.StatusBadRequest)
			return
		}

		if breq.Imp[0].TagID == tag.placementID {
			bid = &openrtb.Bid{
				ID:    "random-id",
				ImpID: breq.Imp[0].ID,
				Price: tag.bid,
				AdM:   tag.content,
			}
			if tag.delay > 0 {
				<-time.After(tag.delay)
			}
		}
	}
	if bid == nil {
		http.Error(w, fmt.Sprintf("Placement ID '%s' not found", breq.Imp[0].TagID), http.StatusInternalServerError)
		return
	}

	resp := openrtb.BidResponse{
		ID:    "a-random-id",
		BidID: "another-random-id",
		Cur:   "USD",
		SeatBid: []openrtb.SeatBid{
			{
				Seat: "FBAN",
				Bid:  []openrtb.Bid{*bid},
			},
		},
	}

	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func GenerateBidRequestForTestData(fbdata bidInfo, url string) (*pbs.PBSRequest, error) {
	pbin := pbs.PBSRequest{
		AdUnits: make([]pbs.AdUnit, len(fbdata.tags)),
	}
	for i, tag := range fbdata.tags {
		pbin.AdUnits[i] = pbs.AdUnit{
			Code:       tag.code,
			MediaTypes: []string{"BANNER"}, // todo set this in fbdata so we can test video setup
			Sizes: []openrtb.Format{
				{
					W: tag.W,
					H: tag.H,
				},
			},
			Bids: []pbs.Bids{
				{
					BidderCode: "audienceNetwork",
					BidID:      fmt.Sprintf("random-id-from-pbjs-%d", i),
					Params:     json.RawMessage(fmt.Sprintf("{\"placementId\": \"%s\"}", tag.placementID)),
				},
			},
			Instl: tag.Instl,
		}
	}

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(pbin)
	if err != nil {
		return nil, err
	}

	req := httptest.NewRequest("POST", url, body)
	req.Header.Add("Referer", fbdata.page)
	req.Header.Add("User-Agent", fbdata.deviceUA)
	req.Header.Add("X-Real-IP", fbdata.deviceIP)

	pc := usersync.ParsePBSCookieFromRequest(req, &config.HostCookie{})
	pc.TrySync("audienceNetwork", fbdata.buyerUID)
	fakewriter := httptest.NewRecorder()
	pc.SetCookieOnResponse(fakewriter, "", 90*24*time.Hour)
	req.Header.Add("Cookie", fakewriter.Header().Get("Set-Cookie"))

	cacheClient, _ := dummycache.New()
	hcc := config.HostCookie{}

	pbReq, err := pbs.ParsePBSRequest(req, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, cacheClient, &hcc)
	return pbReq, err
}

func TestFacebookBasicResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyFacebookServer))
	defer server.Close()

	fbdata = bidInfo{
		partnerID:   12345678,
		domain:      "nytimes.com",
		page:        "https://www.nytimes.com/2017/05/04/movies/guardians-of-the-galaxy-2-review-chris-pratt.html?hpw&rref=movies&action=click&pgtype=Homepage&module=well-region&region=bottom-well&WT.nav=bottom-well&_r=0",
		publisherID: "987654321",
		tags:        make([]tagInfo, 2),
		deviceIP:    "25.91.96.36",
		deviceUA:    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_4) AppleWebKit/603.1.30 (KHTML, like Gecko) Version/10.1 Safari/603.1.30",
		buyerUID:    "need-an-actual-fb-id",
	}
	fbdata.tags[0] = tagInfo{
		code:        "first-tag",
		placementID: fmt.Sprintf("%s_999998888", fbdata.publisherID),
		bid:         1.67,
		W:           300,
		H:           250,
	}
	fbdata.tags[1] = tagInfo{
		code:        "second-tag",
		placementID: fmt.Sprintf("%s_66775544", fbdata.publisherID),
		bid:         3.22,
		W:           300,
		H:           250,
	}

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewFacebookAdapter(&conf, fmt.Sprintf("%d", fbdata.partnerID))
	an.URI = server.URL
	an.nonSecureUri = server.URL

	pbReq, err := GenerateBidRequestForTestData(fbdata, server.URL)

	if err != nil {
		t.Fatalf("ParsePBSRequest failed: %v", err)
	}
	if len(pbReq.Bidders) != 1 {
		t.Fatalf("ParsePBSRequest returned %d bidders instead of 1", len(pbReq.Bidders))
	}
	if pbReq.Bidders[0].BidderCode != "audienceNetwork" {
		t.Fatalf("ParsePBSRequest returned invalid bidder")
	}

	ctx := context.Background()
	bids, err := an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}
	if len(bids) != 2 {
		t.Fatalf("Received %d bids instead of 2", len(bids))
	}
	for _, bid := range bids {
		matched := false
		for _, tag := range fbdata.tags {
			if bid.AdUnitCode == tag.code {
				matched = true
				if bid.BidderCode != "audienceNetwork" {
					t.Errorf("Incorrect BidderCode '%s'", bid.BidderCode)
				}
				if bid.Price != tag.bid {
					t.Errorf("Incorrect bid price '%.2f' expected '%.2f'", bid.Price, tag.bid)
				}
				if bid.Width != tag.W || bid.Height != tag.H {
					t.Errorf("Incorrect bid size %dx%d, expected %dx%d", bid.Width, bid.Height, tag.W, tag.H)
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

	// same test but with one request timing out
	fbdata.tags[0].delay = 20 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	bids, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err != nil {
		// only get an error if everything fails
		t.Fatalf("Should not have gotten an error: %v", err)
	}
	if len(bids) != 1 {
		t.Fatalf("Received %d bids instead of 1", len(bids))
	}
	if bids[0].AdUnitCode != fbdata.tags[1].code {
		t.Fatalf("Didn't receive bid from non-timed out request")
	}
	if bids[0].Price != fbdata.tags[1].bid {
		t.Errorf("Incorrect bid price '%.2f' expected '%.2f'", bids[0].Price, fbdata.tags[1].bid)
	}
}

func TestFacebookInterstitialResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyFacebookServer))
	defer server.Close()

	fbdata = bidInfo{
		partnerID:   12345678,
		domain:      "nytimes.com",
		page:        "https://www.nytimes.com/2017/05/04/movies/guardians-of-the-galaxy-2-review-chris-pratt.html?hpw&rref=movies&action=click&pgtype=Homepage&module=well-region&region=bottom-well&WT.nav=bottom-well&_r=0",
		publisherID: "987654321",
		tags:        make([]tagInfo, 1),
		deviceIP:    "25.91.96.36",
		deviceUA:    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_4) AppleWebKit/603.1.30 (KHTML, like Gecko) Version/10.1 Safari/603.1.30",
		buyerUID:    "need-an-actual-fb-id",
	}
	fbdata.tags[0] = tagInfo{
		code:        "first-tag",
		placementID: fmt.Sprintf("%s_999998888", fbdata.publisherID),
		bid:         1.67,
		W:           300,
		H:           250,
	}

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewFacebookAdapter(&conf, fmt.Sprintf("%d", fbdata.partnerID))
	an.URI = server.URL
	an.nonSecureUri = server.URL

	pbReq, err := GenerateBidRequestForTestData(fbdata, server.URL)
	if err != nil {
		t.Fatalf("ParsePBSRequest failed: %v", err)
	}
	if len(pbReq.Bidders) != 1 {
		t.Fatalf("ParsePBSRequest returned %d bidders instead of 1", len(pbReq.Bidders))
	}
	if pbReq.Bidders[0].BidderCode != "audienceNetwork" {
		t.Fatalf("ParsePBSRequest returned invalid bidder")
	}

	ctx := context.Background()
	bids, err := an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}
	for _, bid := range bids {
		matched := false
		for _, tag := range fbdata.tags {
			if bid.AdUnitCode == tag.code {
				matched = true
				if bid.BidderCode != "audienceNetwork" {
					t.Errorf("Incorrect BidderCode '%s'", bid.BidderCode)
				}
				if bid.Price != tag.bid {
					t.Errorf("Incorrect bid price '%.2f' expected '%.2f'", bid.Price, tag.bid)
				}
				if bid.Width != tag.W || bid.Height != tag.H {
					t.Errorf("Incorrect bid size %dx%d, expected %dx%d", bid.Width, bid.Height, tag.W, tag.H)
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
}

func TestFacebookBannerRequestWithSupportedSizes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyFacebookServer))
	defer server.Close()

	fbdata = bidInfo{
		partnerID:   12345678,
		domain:      "nytimes.com",
		page:        "https://www.nytimes.com/2017/05/04/movies/guardians-of-the-galaxy-2-review-chris-pratt.html?hpw&rref=movies&action=click&pgtype=Homepage&module=well-region&region=bottom-well&WT.nav=bottom-well&_r=0",
		publisherID: "987654321",
		tags:        make([]tagInfo, 3),
		deviceIP:    "25.91.96.36",
		deviceUA:    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_4) AppleWebKit/603.1.30 (KHTML, like Gecko) Version/10.1 Safari/603.1.30",
		buyerUID:    "need-an-actual-fb-id",
	}
	fbdata.tags[0] = tagInfo{
		code:        "first-tag",
		placementID: fmt.Sprintf("%s_999998888", fbdata.publisherID),
		bid:         1.67,
		W:           300,
		H:           250,
	}
	fbdata.tags[1] = tagInfo{
		code:        "second-tag",
		placementID: fmt.Sprintf("%s_948884228", fbdata.publisherID),
		bid:         3.24,
		W:           320,
		H:           50,
	}
	fbdata.tags[2] = tagInfo{
		code:        "third-tag",
		placementID: fmt.Sprintf("%s_122213422", fbdata.publisherID),
		bid:         1.51,
		W:           720,
		H:           90,
	}

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewFacebookAdapter(&conf, fmt.Sprintf("%d", fbdata.partnerID))
	an.URI = server.URL
	an.nonSecureUri = server.URL

	pbReq, err := GenerateBidRequestForTestData(fbdata, server.URL)
	if err != nil {
		t.Fatalf("ParsePBSRequest failed: %v", err)
	}
	if len(pbReq.Bidders) != 1 {
		t.Fatalf("ParsePBSRequest returned %d bidders instead of 1", len(pbReq.Bidders))
	}
	if pbReq.Bidders[0].BidderCode != "audienceNetwork" {
		t.Fatalf("ParsePBSRequest returned invalid bidder")
	}

	ctx := context.Background()
	bids, err := an.Call(ctx, pbReq, pbReq.Bidders[0])
	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}
	for _, bid := range bids {
		matched := false
		for _, tag := range fbdata.tags {
			if bid.AdUnitCode == tag.code {
				matched = true
				if bid.BidderCode != "audienceNetwork" {
					t.Errorf("Incorrect BidderCode '%s'", bid.BidderCode)
				}
				if bid.Price != tag.bid {
					t.Errorf("Incorrect bid price '%.2f' expected '%.2f'", bid.Price, tag.bid)
				}
				if bid.Width != tag.W || bid.Height != tag.H {
					t.Errorf("Incorrect bid size %dx%d, expected %dx%d", bid.Width, bid.Height, tag.W, tag.H)
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
}

func TestGenerateRequestsForFacebook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyFacebookServer))
	defer server.Close()
	// todo: only test for banner now, should add video setup to test that it generates 2 imps per ad unit when PBSAdUnit supports video params
	fbdata = bidInfo{
		partnerID:   12345678,
		domain:      "nytimes.com",
		page:        "https://www.nytimes.com/2017/05/04/movies/guardians-of-the-galaxy-2-review-chris-pratt.html?hpw&rref=movies&action=click&pgtype=Homepage&module=well-region&region=bottom-well&WT.nav=bottom-well&_r=0",
		publisherID: "987654321",
		tags:        make([]tagInfo, 3),
		deviceIP:    "25.91.96.36",
		deviceUA:    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_4) AppleWebKit/603.1.30 (KHTML, like Gecko) Version/10.1 Safari/603.1.30",
		buyerUID:    "need-an-actual-fb-id",
	}
	fbdata.tags[0] = tagInfo{
		code:        "first-tag",
		placementID: fmt.Sprintf("%s_999998888", fbdata.publisherID),
		bid:         1.67,
		W:           300,
		H:           250,
		Instl:       1,
	}
	fbdata.tags[1] = tagInfo{
		code:        "second-tag",
		placementID: fmt.Sprintf("%s_948884228", fbdata.publisherID),
		bid:         3.24,
		W:           320,
		H:           50,
		Instl:       0,
	}
	fbdata.tags[2] = tagInfo{
		code:        "third-tag",
		placementID: fmt.Sprintf("%s_122213422", fbdata.publisherID),
		bid:         1.51,
		W:           720,
		H:           200,
		Instl:       0,
	}

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewFacebookAdapter(&conf, fmt.Sprintf("%d", fbdata.partnerID))
	an.URI = server.URL
	an.nonSecureUri = server.URL

	pbReq, err := GenerateBidRequestForTestData(fbdata, server.URL)
	if err != nil {
		t.Fatalf("ParsePBSRequest failed: %v", err)
	}
	if len(pbReq.Bidders) != 1 {
		t.Fatalf("ParsePBSRequest returned %d bidders instead of 1", len(pbReq.Bidders))
	}
	if pbReq.Bidders[0].BidderCode != "audienceNetwork" {
		t.Fatalf("ParsePBSRequest returned invalid bidder")
	}
	openrtbRequests, err := an.GenerateRequestsForFacebook(pbReq, pbReq.Bidders[0])
	if err != nil {
		t.Fatalf("Generating openrtb requests failed: %v", err)
	}
	if len(openrtbRequests) != 2 {
		t.Fatalf("Should only generate 2 openrtb request")
	}
	if len(openrtbRequests[0].Imp) != 1 {
		t.Fatalf("Should only generate 1 imp per ad unit")
	}
	if len(openrtbRequests[1].Imp) != 1 {
		t.Fatalf("Should only generate 1 imp per ad unit")
	}
	if *openrtbRequests[0].Imp[0].Banner.W != 0 || *openrtbRequests[0].Imp[0].Banner.H != 0 {
		t.Fatalf("Should be generating 0x0 for interstitial type")
	}

	if *openrtbRequests[1].Imp[0].Banner.W != 0 {
		t.Fatalf("Should be passing width 0 for size 320x50")
	}

}
