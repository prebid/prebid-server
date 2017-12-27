package sovrn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/pbs"

	"context"
	"net/http"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/config"
)

func TestJsonSamples(t *testing.T) {
	sovrnAdapter := NewSovrnBidder(new(http.Client), "http://sovrn.com/test/endpoint")
	adapterstest.RunJSONBidderTest(t, "sovrntest", sovrnAdapter)
}

// ----------------------------------------------------------------------------
// Code below this line tests the legacy, non-openrtb code flow. It can be deleted after we
// clean up the existing code and make everything openrtb.

var testSovrnUserId = "SovrnUser123"
var testUserAgent = "user-agent-test"
var testUrl = "http://news.pub/topnews"
var testIp = "123.123.123.123"

func TestSovrnAdapterNames(t *testing.T) {
	adapter := NewSovrnAdapter(adapters.DefaultHTTPAdapterConfig, "http://sovrn/rtb/bid")
	adapters.VerifyStringValue(adapter.Name(), "sovrn", t)
	adapters.VerifyStringValue(adapter.FamilyName(), "sovrn", t)
}

func TestSovrnAdapter_SkipNoCookies(t *testing.T) {
	adapter := NewSovrnAdapter(adapters.DefaultHTTPAdapterConfig, "http://sovrn/rtb/bid")
	adapters.VerifyBoolValue(adapter.SkipNoCookies(), false, t)
}

func TestSovrnOpenRtbRequest(t *testing.T) {
	service := CreateSovrnService(adapters.BidOnTags(""))
	server := service.Server
	ctx := context.TODO()
	req := SampleSovrnRequest(1, t)
	bidder := req.Bidders[0]
	adapter := NewSovrnAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	adapter.Call(ctx, req, bidder)

	adapters.VerifyIntValue(len(service.LastBidRequest.Imp), 1, t)
	adapters.VerifyStringValue(service.LastBidRequest.Imp[0].TagID, "123456", t)
	adapters.VerifyBannerSize(service.LastBidRequest.Imp[0].Banner, 728, 90, t)
	checkHttpRequest(*service.LastHttpRequest, t)
}

func TestSovrnBiddingBehavior(t *testing.T) {
	service := CreateSovrnService(adapters.BidOnTags("123456"))
	server := service.Server
	ctx := context.TODO()
	req := SampleSovrnRequest(1, t)
	bidder := req.Bidders[0]
	adapter := NewSovrnAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	bids, _ := adapter.Call(ctx, req, bidder)

	adapters.VerifyIntValue(len(bids), 1, t)
	adapters.VerifyStringValue(bids[0].AdUnitCode, "div-adunit-1", t)
	adapters.VerifyStringValue(bids[0].BidderCode, "sovrn", t)
	adapters.VerifyStringValue(bids[0].Adm, "<div>This is an Ad</div>", t)
	adapters.VerifyStringValue(bids[0].Creative_id, "Cr-234", t)
	adapters.VerifyIntValue(int(bids[0].Width), 728, t)
	adapters.VerifyIntValue(int(bids[0].Height), 90, t)
	adapters.VerifyIntValue(int(bids[0].Price*100), 210, t)
	checkHttpRequest(*service.LastHttpRequest, t)
}

/**
 * Verify bidding behavior on multiple impressions, some impressions make a bid
 */
func TestSovrntMultiImpPartialBidding(t *testing.T) {
	// setup server endpoint to return bid.
	service := CreateSovrnService(adapters.BidOnTags("123456"))
	server := service.Server
	ctx := context.TODO()
	req := SampleSovrnRequest(2, t)
	bidder := req.Bidders[0]
	adapter := NewSovrnAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	bids, _ := adapter.Call(ctx, req, bidder)
	// two impressions sent.
	// number of bids should be 1
	adapters.VerifyIntValue(len(service.LastBidRequest.Imp), 2, t)
	adapters.VerifyIntValue(len(bids), 1, t)
	adapters.VerifyStringValue(bids[0].AdUnitCode, "div-adunit-1", t)
	checkHttpRequest(*service.LastHttpRequest, t)
}

/**
 * Verify bidding behavior on multiple impressions, all impressions passed back.
 */
func TestSovrnMultiImpAllBid(t *testing.T) {
	// setup server endpoint to return bid.
	service := CreateSovrnService(adapters.BidOnTags("123456,123457"))
	server := service.Server
	ctx := context.TODO()
	req := SampleSovrnRequest(2, t)
	bidder := req.Bidders[0]
	adapter := NewSovrnAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	bids, _ := adapter.Call(ctx, req, bidder)
	// two impressions sent.
	// number of bids should be 1
	adapters.VerifyIntValue(len(service.LastBidRequest.Imp), 2, t)
	adapters.VerifyIntValue(len(bids), 2, t)
	adapters.VerifyStringValue(bids[0].AdUnitCode, "div-adunit-1", t)
	adapters.VerifyStringValue(bids[1].AdUnitCode, "div-adunit-2", t)
	checkHttpRequest(*service.LastHttpRequest, t)
}

func checkHttpRequest(req http.Request, t *testing.T) {
	adapters.VerifyStringValue(req.Header.Get("Accept-Language"), "murican", t)
	var cookie, _ = req.Cookie("ljt_reader")
	adapters.VerifyStringValue((*cookie).Value, testSovrnUserId, t)
	adapters.VerifyStringValue(req.Header.Get("User-Agent"), testUserAgent, t)
	adapters.VerifyStringValue(req.Header.Get("Content-Type"), "application/json", t)
	adapters.VerifyStringValue(req.Header.Get("X-Forwarded-For"), testIp, t)
	adapters.VerifyStringValue(req.Header.Get("DNT"), "0", t)
}

func SampleSovrnRequest(numberOfImpressions int, t *testing.T) *pbs.PBSRequest {
	device := openrtb.Device{
		Language: "murican",
	}

	user := openrtb.User{
		ID: testSovrnUserId,
	}

	req := pbs.PBSRequest{
		AccountID: "1",
		AdUnits:   make([]pbs.AdUnit, 2),
		Device:    &device,
		User:      &user,
	}

	tagID := 123456

	for i := 0; i < numberOfImpressions; i++ {
		req.AdUnits[i] = pbs.AdUnit{
			Code: fmt.Sprintf("div-adunit-%d", i+1),
			Sizes: []openrtb.Format{
				{
					W: 728,
					H: 90,
				},
			},
			Bids: []pbs.Bids{
				{
					BidderCode: "sovrn",
					BidID:      fmt.Sprintf("Bid-%d", i+1),
					Params:     json.RawMessage(fmt.Sprintf("{\"tagid\": %d }", tagID+i)),
				},
			},
		}

	}

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(req)
	if err != nil {
		t.Fatalf("Error when serializing request")
	}

	httpReq := httptest.NewRequest("POST", CreateSovrnService(adapters.BidOnTags("")).Server.URL, body)
	httpReq.Header.Add("Referer", testUrl)
	httpReq.Header.Add("User-Agent", testUserAgent)
	httpReq.Header.Add("X-Forwarded-For", testIp)
	pc := pbs.ParsePBSCookieFromRequest(httpReq, &config.Cookie{})
	pc.TrySync("sovrn", testSovrnUserId)
	fakewriter := httptest.NewRecorder()
	pc.SetCookieOnResponse(fakewriter, "")
	httpReq.Header.Add("Cookie", fakewriter.Header().Get("Set-Cookie"))
	// parse the http request
	cacheClient, _ := dummycache.New()
	hcs := pbs.HostCookieSettings{}

	parsedReq, err := pbs.ParsePBSRequest(httpReq, cacheClient, &hcs)
	if err != nil {
		t.Fatalf("Error when parsing request: %v", err)
	}
	return parsedReq

}

func TestNoContentResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	ctx := context.TODO()
	req := SampleSovrnRequest(1, t)
	bidder := req.Bidders[0]
	adapter := NewSovrnAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	_, err := adapter.Call(ctx, req, bidder)

	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}

}

func TestNotFoundResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	ctx := context.TODO()
	req := SampleSovrnRequest(1, t)
	bidder := req.Bidders[0]
	adapter := NewSovrnAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	_, err := adapter.Call(ctx, req, bidder)

	adapters.VerifyStringValue(err.Error(), "HTTP status 404; body: ", t)

}

func CreateSovrnService(tagsToBid map[string]bool) adapters.OrtbMockService {
	service := adapters.OrtbMockService{}
	var lastBidRequest openrtb.BidRequest
	var lastHttpReq http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastHttpReq = *r
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
		lastBidRequest = breq
		var bids []openrtb.Bid
		for i, imp := range breq.Imp {
			if tagsToBid[imp.TagID] {
				bids = append(bids, adapters.SampleBid(imp.Banner.W, imp.Banner.H, imp.ID, i+1))
			}
		}

		// serialize the bids to openrtb.BidResponse
		js, _ := json.Marshal(openrtb.BidResponse{
			SeatBid: []openrtb.SeatBid{
				{
					Bid: bids,
				},
			},
		})
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}))

	service.Server = server
	service.LastBidRequest = &lastBidRequest
	service.LastHttpRequest = &lastHttpReq

	return service
}
