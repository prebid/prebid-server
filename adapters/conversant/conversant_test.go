package conversant

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"io/ioutil"

	"net/http"
	"net/http/httptest"

	"encoding/json"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/pbs"
)

// Constants

const ExpectedSiteID string = "12345"
const ExpectedDisplayManager string = "prebid-s2s"
const ExpectedBuyerUID string = "AQECT_o7M1FLbQJK8QFmAQEBAQE"
const ExpectedNURL string = "http://test.dotomi.com"
const ExpectedAdM string = "<img src=\"test.jpg\"/>"
const ExpectedCrID string = "98765"

const EmptyParam = "{}"
const DefaultParam = `{"site_id": "12345"}`

/**
* Test properties of Adapter interface
 */

func TestConversantProperties(t *testing.T) {
	an := NewConversantAdapter(adapters.DefaultHTTPAdapterConfig, "someUrl", "usersync", "localhost")

	assertNotEqual(t, an.Name(), "", "Missing name")
	assertNotEqual(t, an.FamilyName(), "", "Missing family name")
	assertFalse(t, an.SkipNoCookies(), "SkipNoCookies should be false")
}

/**
* Test empty bid requests
 */

func TestConversantEmptyBid(t *testing.T) {
	an := NewConversantAdapter(adapters.DefaultHTTPAdapterConfig, "someUrl", "usersync", "localhost")

	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	assertTrue(t, err != nil, "No error received for an invalid request")
}

/**
* Test required parameters, which is just the site id for now
 */

func TestConversantRequiredParameters(t *testing.T) {
	an := NewConversantAdapter(adapters.DefaultHTTPAdapterConfig, "someUrl", "usersync", "localhost")

	ctx := context.TODO()
	pbReq := ParseRequest(CreateRequest(EmptyParam), t)
	pbBidder := pbReq.Bidders[0]

	_, err := an.Call(ctx, pbReq, pbBidder)
	assertTrue(t, err != nil, "Failed to catch missing site id")
}

/**
* Test handling of 404
 */

func TestConversantBadStatus(t *testing.T) {
	// Create a test http server that returns after 2 milliseconds

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}),
	)
	defer server.Close()

	// Create a adapter to test

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewConversantAdapter(&conf, server.URL, "usersync", "localhost")

	ctx := context.TODO()
	pbReq := ParseRequest(CreateRequest(DefaultParam), t)
	pbBidder := pbReq.Bidders[0]

	_, err := an.Call(ctx, pbReq, pbBidder)
	assertTrue(t, err != nil, "Failed to catch 404 error")
}

/**
* Test handling of HTTP timeout
 */

func TestConversantTimeout(t *testing.T) {
	// Create a test http server that returns after 2 milliseconds

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-time.After(2 * time.Millisecond)
		}),
	)
	defer server.Close()

	// Create a adapter to test

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewConversantAdapter(&conf, server.URL, "usersync", "localhost")

	// Create a context that expires before http returns

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	// Create a basic request
	pbReq := ParseRequest(CreateRequest(DefaultParam), t)
	pbBidder := pbReq.Bidders[0]

	// Attempt to process the request, which should hit a timeout
	// immediately

	_, err := an.Call(ctx, pbReq, pbBidder)
	if err == nil || err != context.DeadlineExceeded {
		t.Fatalf("No timeout recevied for timed out request: %v", err)
	}
}

/**
* Test handling of 204
 */

func TestConversantNoBid(t *testing.T) {
	// Create a test http server that returns after 2 milliseconds

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
		}),
	)
	defer server.Close()

	// Create a adapter to test

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewConversantAdapter(&conf, server.URL, "usersync", "localhost")

	ctx := context.TODO()
	pbReq := ParseRequest(CreateRequest(DefaultParam), t)
	pbBidder := pbReq.Bidders[0]

	resp, err := an.Call(ctx, pbReq, pbBidder)
	if resp != nil || err != nil {
		t.Fatal("Failed to handle emtpy bid")
	}
}

/**
* Check user sync information
 */

func TestConversantUserSyncInfo(t *testing.T) {
	an := NewConversantAdapter(adapters.DefaultHTTPAdapterConfig, "someUrl", "usersync?rurl=", "localhost")

	if !strings.HasSuffix(an.usersyncInfo.URL, "?rurl=localhost%2Fsetuid%3Fbidder%3Dconversant%26uid%3D") {
		t.Fatalf("bad user sync url: %s", an.usersyncInfo.URL)
	}

	if an.usersyncInfo.Type != "redirect" {
		t.Fatalf("user sync type should be redirect: %s", an.usersyncInfo.Type)
	}

	if an.usersyncInfo.SupportCORS != false {
		t.Fatalf("user sync should not support CORS")
	}
}

/**
* Verify an outgoing openrtp request is created correctly
 */

func TestConversantRequestDefault(t *testing.T) {
	server, lastReq := CreateServer([]float64{})
	if server == nil {
		t.Fatal("server not created")
	}

	defer server.Close()

	// Create a adapter to test

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewConversantAdapter(&conf, server.URL, "usersync", "localhost")

	ctx := context.TODO()
	pbReq := ParseRequest(CreateRequest(DefaultParam), t)
	pbBidder := pbReq.Bidders[0]

	_, err := an.Call(ctx, pbReq, pbBidder)
	if err != nil {
		t.Fatal("Failed to retrieve bids")
	}

	assertEqual(t, len(lastReq.Imp), 1, "Request number of impressions")
	assertEqual(t, lastReq.Imp[0].DisplayManager, ExpectedDisplayManager, "Request display manager value")
	assertEqual(t, lastReq.Site.ID, ExpectedSiteID, "Request site id")
	assertEqual(t, int(lastReq.Site.Mobile), 0, "Request site mobile flag")
	assertEqual(t, lastReq.User.BuyerUID, ExpectedBuyerUID, "Request buyeruid")
	assertTrue(t, lastReq.Imp[0].Video == nil, "Request video should be nil")
	assertEqual(t, int(*lastReq.Imp[0].Secure), 0, "Request secure")
	assertEqual(t, lastReq.Imp[0].BidFloor, 0.0, "Request bid floor")
	assertEqual(t, lastReq.Imp[0].TagID, "", "Request tag id")
	assertTrue(t, lastReq.Imp[0].Banner.Pos == nil, "Request pos")
	assertEqual(t, int(*lastReq.Imp[0].Banner.W), 300, "Request width")
	assertEqual(t, int(*lastReq.Imp[0].Banner.H), 250, "Request height")
}

/**
* Verify an outgoing openrtp request with additional conversant parameters is
* processed correctly
 */

func TestConversantRequest(t *testing.T) {
	server, lastReq := CreateServer([]float64{})
	if server == nil {
		t.Fatal("server not created")
	}

	defer server.Close()

	// Create a adapter to test

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewConversantAdapter(&conf, server.URL, "usersync", "localhost")

	param := `{ "site_id": "12345",
		"secure": true,
		"tag_id": "top",
		"position": 2,
		"bidfloor": 1.01,
		"mobile": 1 }`

	ctx := context.TODO()
	pbReq := ParseRequest(CreateRequest(param), t)
	pbBidder := pbReq.Bidders[0]

	_, err := an.Call(ctx, pbReq, pbBidder)
	if err != nil {
		t.Fatal("Failed to retrieve bids")
	}

	assertEqual(t, len(lastReq.Imp), 1, "Request number of impressions")
	assertEqual(t, lastReq.Imp[0].DisplayManager, ExpectedDisplayManager, "Request display manager value")
	assertEqual(t, lastReq.Site.ID, ExpectedSiteID, "Request site id")
	assertEqual(t, int(lastReq.Site.Mobile), 1, "Request site mobile flag")
	assertEqual(t, lastReq.User.BuyerUID, ExpectedBuyerUID, "Request buyeruid")
	assertTrue(t, lastReq.Imp[0].Video == nil, "Request video should be nil")
	assertEqual(t, int(*lastReq.Imp[0].Secure), 1, "Request secure")
	assertEqual(t, lastReq.Imp[0].BidFloor, 1.01, "Request bid floor")
	assertEqual(t, lastReq.Imp[0].TagID, "top", "Request tag id")
	assertEqual(t, int(*lastReq.Imp[0].Banner.Pos), 2, "Request pos")
	assertEqual(t, int(*lastReq.Imp[0].Banner.W), 300, "Request width")
	assertEqual(t, int(*lastReq.Imp[0].Banner.H), 250, "Request height")
}

/**
* Verify openrtp responses are converted correctly
 */

func TestConversantResponse(t *testing.T) {
	prices := []float64{0.01, 0.0, 2.01}
	server, lastReq := CreateServer(prices)
	if server == nil {
		t.Fatal("server not created")
	}

	defer server.Close()

	// Create a adapter to test

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewConversantAdapter(&conf, server.URL, "usersync", "localhost")

	param := `{ "site_id": "12345",
		   "secure": true,
		   "tag_id": "top",
		   "position": 2,
		   "bidfloor": 1.01,
		   "mobile" : 1}`

	ctx := context.TODO()
	pbReq := ParseRequest(CreateRequest(param, param, param), t)
	pbBidder := pbReq.Bidders[0]

	resp, err := an.Call(ctx, pbReq, pbBidder)
	if err != nil {
		t.Fatal("Failed to retrieve bids")
	}

	prices, imps := FilterZeroPrices(prices, lastReq.Imp)

	assertEqual(t, len(resp), len(prices), "Bad number of reponses")

	for i, bid := range resp {
		assertEqual(t, bid.Price, prices[i], "Bad price in response")
		assertEqual(t, bid.AdUnitCode, imps[i].ID, "Bad bid id in response")

		if bid.Price > 0 {
			assertEqual(t, bid.Adm, ExpectedAdM, "Bad ad markup in response")
			assertEqual(t, bid.NURL, ExpectedNURL, "Bad notification url in response")
			assertEqual(t, bid.Creative_id, ExpectedCrID, "Bad creative id in response")
			assertEqual(t, bid.Width, *imps[i].Banner.W, "Bad width in response")
			assertEqual(t, bid.Height, *imps[i].Banner.H, "Bad height in response")
		}
	}
}

/**
 * Test video request
 */

func TestConversantBasicVideoRequest(t *testing.T) {
	server, lastReq := CreateServer([]float64{})
	if server == nil {
		t.Fatal("server not created")
	}

	defer server.Close()

	// Create a adapter to test

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewConversantAdapter(&conf, server.URL, "usersync", "localhost")

	param := `{ "site_id": "12345",
		   "tag_id": "bottom left",
		   "position": 3,
		   "bidfloor": 1.01 }`

	ctx := context.TODO()
	pbReq := CreateRequest(param)
	pbReq = ConvertToVideoRequest(pbReq, t)
	pbReq = ParseRequest(pbReq, t)
	pbBidder := pbReq.Bidders[0]

	_, err := an.Call(ctx, pbReq, pbBidder)
	if err != nil {
		t.Fatal("Failed to retrieve bids")
	}

	assertEqual(t, len(lastReq.Imp), 1, "Request number of impressions")
	assertEqual(t, lastReq.Imp[0].DisplayManager, ExpectedDisplayManager, "Request display manager value")
	assertEqual(t, lastReq.Site.ID, ExpectedSiteID, "Request site id")
	assertEqual(t, int(lastReq.Site.Mobile), 0, "Request site mobile flag")
	assertEqual(t, lastReq.User.BuyerUID, ExpectedBuyerUID, "Request buyeruid")
	assertTrue(t, lastReq.Imp[0].Banner == nil, "Request banner should be nil")
	assertEqual(t, int(*lastReq.Imp[0].Secure), 0, "Request secure")
	assertEqual(t, lastReq.Imp[0].BidFloor, 1.01, "Request bid floor")
	assertEqual(t, lastReq.Imp[0].TagID, "bottom left", "Request tag id")
	assertEqual(t, int(*lastReq.Imp[0].Video.Pos), 3, "Request pos")
	assertEqual(t, int(lastReq.Imp[0].Video.W), 300, "Request width")
	assertEqual(t, int(lastReq.Imp[0].Video.H), 250, "Request height")

	assertEqual(t, len(lastReq.Imp[0].Video.MIMEs), 1, "Request video MIMEs entries")
	assertEqual(t, lastReq.Imp[0].Video.MIMEs[0], "video/mp4", "Requst video MIMEs type")
	assertTrue(t, lastReq.Imp[0].Video.Protocols == nil, "Request video protocols")
	assertEqual(t, lastReq.Imp[0].Video.MaxDuration, int64(0), "Request video 0 max duration")
	assertTrue(t, lastReq.Imp[0].Video.API == nil, "Request video api should be nil")
}

func TestConversantVideoRequestWithParams(t *testing.T) {
	server, lastReq := CreateServer([]float64{})
	if server == nil {
		t.Fatal("server not created")
	}

	defer server.Close()

	// Create a adapter to test

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewConversantAdapter(&conf, server.URL, "usersync", "localhost")

	param := `{ "site_id": "12345",
		   "tag_id": "bottom left",
		   "position": 3,
		   "bidfloor": 1.01, 
		   "mimes": ["video/x-ms-wmv"],
		   "protocols": [1, 2],
		   "api": [1, 2],
		   "maxduration": 90 }`

	ctx := context.TODO()
	pbReq := CreateRequest(param)
	pbReq = ConvertToVideoRequest(pbReq, t)
	pbReq = ParseRequest(pbReq, t)
	pbBidder := pbReq.Bidders[0]

	_, err := an.Call(ctx, pbReq, pbBidder)
	if err != nil {
		t.Fatal("Failed to retrieve bids")
	}

	assertEqual(t, len(lastReq.Imp), 1, "Request number of impressions")
	assertEqual(t, lastReq.Imp[0].DisplayManager, ExpectedDisplayManager, "Request display manager value")
	assertEqual(t, lastReq.Site.ID, ExpectedSiteID, "Request site id")
	assertEqual(t, int(lastReq.Site.Mobile), 0, "Request site mobile flag")
	assertEqual(t, lastReq.User.BuyerUID, ExpectedBuyerUID, "Request buyeruid")
	assertTrue(t, lastReq.Imp[0].Banner == nil, "Request banner should be nil")
	assertEqual(t, int(*lastReq.Imp[0].Secure), 0, "Request secure")
	assertEqual(t, lastReq.Imp[0].BidFloor, 1.01, "Request bid floor")
	assertEqual(t, lastReq.Imp[0].TagID, "bottom left", "Request tag id")
	assertEqual(t, int(*lastReq.Imp[0].Video.Pos), 3, "Request pos")
	assertEqual(t, int(lastReq.Imp[0].Video.W), 300, "Request width")
	assertEqual(t, int(lastReq.Imp[0].Video.H), 250, "Request height")

	assertEqual(t, len(lastReq.Imp[0].Video.MIMEs), 1, "Request video MIMEs entries")
	assertEqual(t, lastReq.Imp[0].Video.MIMEs[0], "video/x-ms-wmv", "Requst video MIMEs type")
	assertEqual(t, len(lastReq.Imp[0].Video.Protocols), 2, "Request video protocols")
	assertEqual(t, lastReq.Imp[0].Video.Protocols[0], openrtb.Protocol(1), "Request video protocols 1")
	assertEqual(t, lastReq.Imp[0].Video.Protocols[1], openrtb.Protocol(2), "Request video protocols 2")
	assertEqual(t, lastReq.Imp[0].Video.MaxDuration, int64(90), "Request video 0 max duration")
	assertEqual(t, len(lastReq.Imp[0].Video.API), 2, "Request video api should be nil")
	assertEqual(t, lastReq.Imp[0].Video.API[0], openrtb.APIFramework(1), "Request video api 1")
	assertEqual(t, lastReq.Imp[0].Video.API[1], openrtb.APIFramework(2), "Request video api 2")
}

func TestConversantVideoResponse(t *testing.T) {
	prices := []float64{0.01, 0.0, 2.01}
	server, lastReq := CreateServer(prices)
	if server == nil {
		t.Fatal("server not created")
	}

	defer server.Close()

	// Create a adapter to test

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewConversantAdapter(&conf, server.URL, "usersync", "localhost")

	param := `{ "site_id": "12345",
		   "secure": true,
		   "tag_id": "top",
		   "position": 2,
		   "bidfloor": 1.01,
		   "mobile" : 1}`

	ctx := context.TODO()
	pbReq := ParseRequest(CreateRequest(param, param, param), t)
	pbReq = ConvertToVideoRequest(pbReq, t)
	pbReq = ParseRequest(pbReq, t)
	pbBidder := pbReq.Bidders[0]

	resp, err := an.Call(ctx, pbReq, pbBidder)
	if err != nil {
		t.Fatal("Failed to retrieve bids")
	}

	prices, imps := FilterZeroPrices(prices, lastReq.Imp)

	assertEqual(t, len(resp), len(prices), "Bad number of reponses")

	for i, bid := range resp {
		assertEqual(t, bid.Price, prices[i], "Bad price in response")
		assertEqual(t, bid.AdUnitCode, imps[i].ID, "Bad bid id in response")

		if bid.Price > 0 {
			assertEqual(t, bid.Adm, "", "Bad ad markup in response")
			assertEqual(t, bid.NURL, ExpectedAdM, "Bad notification url in response")
			assertEqual(t, bid.Creative_id, ExpectedCrID, "Bad creative id in response")
			assertEqual(t, bid.Width, imps[i].Video.W, "Bad width in response")
			assertEqual(t, bid.Height, imps[i].Video.H, "Bad height in response")
		}
	}
}

/**
 * Helpers to create a banner and video requests
 */

func CreateRequest(params ...string) *pbs.PBSRequest {
	num := len(params)

	req := pbs.PBSRequest{
		Tid:       "t-000",
		AccountID: "1",
		AdUnits:   make([]pbs.AdUnit, num),
	}

	for i := 0; i < num; i++ {
		req.AdUnits[i] = pbs.AdUnit{
			Code: fmt.Sprintf("au-%03d", i),
			Sizes: []openrtb.Format{
				{
					W: 300,
					H: 250,
				},
			},
			Bids: []pbs.Bids{
				{
					BidderCode: "conversant",
					BidID:      fmt.Sprintf("b-%03d", i),
					Params:     json.RawMessage(params[i]),
				},
			},
		}
	}

	return &req
}

func ConvertToVideoRequest(req *pbs.PBSRequest, t *testing.T, videoParams ...string) *pbs.PBSRequest {
	for i := 0; i < len(req.AdUnits); i++ {
		video := pbs.PBSVideo{}
		if i < len(videoParams) {
			err := json.Unmarshal([]byte(videoParams[i]), &video)
			if err != nil {
				t.Fatalf("Failed to parse video param: %v", err)
			}
		}

		if video.Mimes == nil {
			video.Mimes = []string{"video/mp4"}
		}

		req.AdUnits[i].Video = video
		req.AdUnits[i].MediaTypes = []string{"video"}
	}

	return req
}

func ParseRequest(req *pbs.PBSRequest, t *testing.T) *pbs.PBSRequest {
	body := new(bytes.Buffer)
	json.NewEncoder(body).Encode(req)

	// Need to pass the conversant user id thru uid cookie

	httpReq := httptest.NewRequest("POST", "/foo", body)
	cookie := pbs.NewPBSCookie()
	cookie.TrySync("conversant", ExpectedBuyerUID)
	httpReq.Header.Set("Cookie", cookie.ToHTTPCookie().String())
	httpReq.Header.Add("Referer", "http://example.com")
	cache, _ := dummycache.New()
	hcs := pbs.HostCookieSettings{}

	parsedReq, err := pbs.ParsePBSRequest(httpReq, cache, &hcs)

	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	return parsedReq
}

/**
* Helper to create a test http server that receives and generate openrtb requests and responses
 */

func CreateServer(prices []float64) (*httptest.Server, *openrtb.BidRequest) {
	var lastBidRequest openrtb.BidRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var bidReq openrtb.BidRequest
		var price float64
		var bids []openrtb.Bid
		var bid openrtb.Bid

		err = json.Unmarshal(body, &bidReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		lastBidRequest = bidReq

		for i, imp := range bidReq.Imp {
			if i < len(prices) {
				price = prices[i]
			} else {
				price = 0
			}

			if price > 0 {
				bid = openrtb.Bid{
					ID:    imp.ID,
					ImpID: imp.ID,
					Price: price,
					NURL:  ExpectedNURL,
					AdM:   ExpectedAdM,
					CrID:  ExpectedCrID,
				}

				if imp.Banner != nil {
					bid.W = *imp.Banner.W
					bid.H = *imp.Banner.H
				} else if imp.Video != nil {
					bid.W = imp.Video.W
					bid.H = imp.Video.H
				}
			} else {
				bid = openrtb.Bid{
					ID:    imp.ID,
					ImpID: imp.ID,
					Price: 0,
				}
			}

			bids = append(bids, bid)
		}

		if len(bids) == 0 {
			w.WriteHeader(http.StatusNoContent)
		} else {
			js, _ := json.Marshal(openrtb.BidResponse{
				ID: bidReq.ID,
				SeatBid: []openrtb.SeatBid{
					{
						Bid: bids,
					},
				},
			})

			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
		}
	}),
	)

	return server, &lastBidRequest
}

func FilterZeroPrices(prices []float64, imps []openrtb.Imp) ([]float64, []openrtb.Imp) {
	prices2 := make([]float64, 0)
	imps2 := make([]openrtb.Imp, 0)

	for i, _ := range prices {
		if prices[i] > 0 {
			prices2 = append(prices2, prices[i])
			imps2 = append(imps2, imps[i])
		}
	}

	return prices2, imps2
}

func assertEqual(t *testing.T, actual interface{}, expected interface{}, msg string) {
	if expected != actual {
		msg = fmt.Sprintf("%s: act(%v) != exp(%v)", msg, actual, expected)
		t.Fatal(msg)
	}
}

func assertNotEqual(t *testing.T, actual interface{}, expected interface{}, msg string) {
	if expected == actual {
		msg = fmt.Sprintf("%s: act(%v) == exp(%v)", msg, actual, expected)
		t.Fatal(msg)
	}
}

func assertTrue(t *testing.T, val bool, msg string) {
	if val == false {
		msg = fmt.Sprintf("%s: is false but should be true", msg)
		t.Fatal(msg)
	}
}

func assertFalse(t *testing.T, val bool, msg string) {
	if val == true {
		msg = fmt.Sprintf("%s: is true but should be false", msg)
		t.Fatal(msg)
	}
}
