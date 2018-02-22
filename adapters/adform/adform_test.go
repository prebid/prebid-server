package adform

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/prebid/prebid-server/openrtb_ext"
)

type aTagInfo struct {
	mid  uint32
	code string

	price   float64
	content string
	dealId  string
}

type aBidInfo struct {
	deviceIP string
	deviceUA string
	tags     []aTagInfo
	referrer string
	width    uint64
	height   uint64
	tid      string
	buyerUID string
	secure   bool
	delay    time.Duration
}

var adformTestData aBidInfo

// Legacy auction tests

func DummyAdformServer(w http.ResponseWriter, r *http.Request) {
	errorString := assertAdformServerRequest(adformTestData, r)
	if errorString != nil {
		http.Error(w, *errorString, http.StatusInternalServerError)
		return
	}

	if adformTestData.delay > 0 {
		<-time.After(adformTestData.delay)
	}

	adformServerResponse, err := createAdformServerResponse(adformTestData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(adformServerResponse)
}

func createAdformServerResponse(testData aBidInfo) ([]byte, error) {
	bids := []adformBid{
		{
			ResponseType: "banner",
			Banner:       testData.tags[0].content,
			Price:        testData.tags[0].price,
			Currency:     "USD",
			Width:        testData.width,
			Height:       testData.height,
			DealId:       testData.tags[0].dealId,
		},
		{},
		{
			ResponseType: "banner",
			Banner:       testData.tags[2].content,
			Price:        testData.tags[2].price,
			Currency:     "USD",
			Width:        testData.width,
			Height:       testData.height,
			DealId:       testData.tags[2].dealId,
		},
	}
	adformServerResponse, err := json.Marshal(bids)
	return adformServerResponse, err
}

func TestAdformBasicResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyAdformServer))
	defer server.Close()

	adapter, ctx, prebidRequest := initTestData(server, t)

	bids, err := adapter.Call(ctx, prebidRequest, prebidRequest.Bidders[0])

	if err != nil {
		t.Fatalf("Should not have gotten adapter error: %v", err)
	}
	if len(bids) != 2 {
		t.Fatalf("Received %d bids instead of 2", len(bids))
	}
	for _, bid := range bids {
		matched := false
		for _, tag := range adformTestData.tags {
			if bid.AdUnitCode == tag.code {
				matched = true
				if bid.BidderCode != "adform" {
					t.Errorf("Incorrect BidderCode '%s'", bid.BidderCode)
				}
				if bid.Price != tag.price {
					t.Errorf("Incorrect bid price '%.2f' expected '%.2f'", bid.Price, tag.price)
				}
				if bid.Width != adformTestData.width || bid.Height != adformTestData.height {
					t.Errorf("Incorrect bid size %dx%d, expected %dx%d", bid.Width, bid.Height, adformTestData.width, adformTestData.height)
				}
				if bid.Adm != tag.content {
					t.Errorf("Incorrect bid markup '%s' expected '%s'", bid.Adm, tag.content)
				}
				if bid.DealId != tag.dealId {
					t.Errorf("Incorrect deal id '%s' expected '%s'", bid.DealId, tag.dealId)
				}
			}
		}
		if !matched {
			t.Errorf("Received bid for unknown ad unit '%s'", bid.AdUnitCode)
		}
	}

	// same test but with request timing out
	adformTestData.delay = 5 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	bids, err = adapter.Call(ctx, prebidRequest, prebidRequest.Bidders[0])
	if err == nil {
		t.Fatalf("Should have gotten a timeout error: %v", err)
	}
}

func initTestData(server *httptest.Server, t *testing.T) (*AdformAdapter, context.Context, *pbs.PBSRequest) {
	adformTestData = aBidInfo{
		deviceIP: "111.111.111.111",
		deviceUA: "Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_1 like Mac OS X) AppleWebKit/603.1.30 (KHTML, like Gecko) Mobile/14E8301",
		tags:     make([]aTagInfo, 3),
		referrer: "http://test.com",
		width:    200,
		height:   300,
		tid:      "transaction-id",
		buyerUID: "user-id",
		secure:   false,
	}
	adformTestData.tags[0] = aTagInfo{mid: 32344, code: "code1", price: 1.23, content: "banner-content1", dealId: "dealId1"}
	adformTestData.tags[1] = aTagInfo{mid: 32345, code: "code2"} // no bid for ad unit
	adformTestData.tags[2] = aTagInfo{mid: 32346, code: "code3", price: 1.24, content: "banner-content2", dealId: "dealId2"}

	// prepare adapter
	conf := *adapters.DefaultHTTPAdapterConfig
	adapter := NewAdformAdapter(&conf, "adx.adform.net/adx")
	adapter.URI = server.URL

	prebidRequest := preparePrebidRequest(server.URL, t)
	ctx := context.TODO()

	return adapter, ctx, prebidRequest
}

func preparePrebidRequest(serverUrl string, t *testing.T) *pbs.PBSRequest {
	body := preparePrebidRequestBody(adformTestData, t)
	prebidHttpRequest := httptest.NewRequest("POST", serverUrl, body)
	prebidHttpRequest.Header.Add("User-Agent", adformTestData.deviceUA)
	prebidHttpRequest.Header.Add("Referer", adformTestData.referrer)
	prebidHttpRequest.Header.Add("X-Real-IP", adformTestData.deviceIP)

	pbsCookie := pbs.ParsePBSCookieFromRequest(prebidHttpRequest, &config.Cookie{})
	pbsCookie.TrySync("adform", adformTestData.buyerUID)
	fakeWriter := httptest.NewRecorder()
	pbsCookie.SetCookieOnResponse(fakeWriter, "", time.Minute)
	prebidHttpRequest.Header.Add("Cookie", fakeWriter.Header().Get("Set-Cookie"))

	cacheClient, _ := dummycache.New()
	r, err := pbs.ParsePBSRequest(prebidHttpRequest, cacheClient, &pbs.HostCookieSettings{})
	if err != nil {
		t.Fatalf("ParsePBSRequest failed: %v", err)
	}
	if len(r.Bidders) != 1 {
		t.Fatalf("ParsePBSRequest returned %d bidders instead of 1", len(r.Bidders))
	}
	if r.Bidders[0].BidderCode != "adform" {
		t.Fatalf("ParsePBSRequest returned invalid bidder")
	}
	return r
}

func preparePrebidRequestBody(requestData aBidInfo, t *testing.T) *bytes.Buffer {
	prebidRequest := pbs.PBSRequest{
		AdUnits: make([]pbs.AdUnit, 3),
		Device: &openrtb.Device{
			UA: requestData.deviceUA,
			IP: requestData.deviceIP,
		},
		Tid:    requestData.tid,
		Secure: 0,
	}
	for i, tag := range requestData.tags {
		prebidRequest.AdUnits[i] = pbs.AdUnit{
			Code: tag.code,
			Sizes: []openrtb.Format{
				{
					W: requestData.width,
					H: requestData.height,
				},
			},
			Bids: []pbs.Bids{
				{
					BidderCode: "adform",
					BidID:      fmt.Sprintf("random-id-from-pbjs-%d", i),
					Params:     json.RawMessage(fmt.Sprintf("{\"mid\": %d}", tag.mid)),
				},
			},
		}
	}
	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(prebidRequest)
	if err != nil {
		t.Fatalf("Json encoding failed: %v", err)
	}
	fmt.Println("body", body)
	return body
}

// OpenRTB auction tests

func TestOpenRTBRequest(t *testing.T) {
	bidder := new(AdformAdapter)
	bidder.URI = "http://adx.adform.net"
	testData := createTestData()
	request := createOpenRtbRequest(testData)

	httpRequests, errs := bidder.MakeRequests(request)

	if len(errs) > 0 {
		t.Errorf("Got unexpected errors while building HTTP requests: %v", errs)
	}
	if len(httpRequests) != 1 {
		t.Fatalf("Unexpected number of HTTP requests. Got %d. Expected %d", len(httpRequests), 1)
	}

	r, err := http.NewRequest(httpRequests[0].Method, httpRequests[0].Uri, bytes.NewReader(httpRequests[0].Body))
	if err != nil {
		t.Fatalf("Unexpected request. Got %v", err)
	}
	r.Header = httpRequests[0].Headers

	errorString := assertAdformServerRequest(*testData, r)
	if errorString != nil {
		t.Errorf("Request error: %s", *errorString)
	}
}

func TestOpenRTBIncorrectRequest(t *testing.T) {
	bidder := new(AdformAdapter)
	request := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{
			{ID: "video-not-supported", Video: &openrtb.Video{}, Ext: openrtb.RawJSON(`{"bidder": { "mid": "32344" }}`)},
			{ID: "audio-not-supported", Audio: &openrtb.Audio{}, Ext: openrtb.RawJSON(`{"bidder": { "mid": "32344" }}`)},
			{ID: "native-not-supported", Native: &openrtb.Native{}, Ext: openrtb.RawJSON(`{"bidder": { "mid": "32344" }}`)},
			{ID: "incorrect-bidder-field", Ext: openrtb.RawJSON(`{"bidder1": { "mid": "32344" }}`)},
			{ID: "incorrect-adform-params", Ext: openrtb.RawJSON(`{"bidder": { : "33" }}`)},
			{ID: "mid-integer", Ext: openrtb.RawJSON(`{"bidder": { "mid": 1.234 }}`)},
			{ID: "mid-greater-then-zero", Ext: openrtb.RawJSON(`{"bidder": { "mid": -1 }}`)},
		},
		Device: &openrtb.Device{UA: "ua", IP: "ip"},
		User:   &openrtb.User{BuyerUID: "buyerUID"},
	}

	httpRequests, errs := bidder.MakeRequests(request)

	if len(errs) != len(request.Imp) {
		t.Errorf("%d Imp objects should have errors. but was %d errors", len(request.Imp), len(errs))
	}
	if len(httpRequests) != 0 {
		t.Fatalf("All Imp objects have errors, but requests count: %d. Expected %d", len(httpRequests), 0)
	}
}

func createTestData() *aBidInfo {
	testData := &aBidInfo{
		deviceIP: "111.111.111.111",
		deviceUA: "Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_1 like Mac OS X) AppleWebKit/603.1.30 (KHTML, like Gecko) Mobile/14E8301",
		referrer: "http://test.com",
		tid:      "transaction-id",
		buyerUID: "user-id",
		tags: []aTagInfo{
			{mid: 32344, code: "code1", price: 1.23, content: "banner-content1", dealId: "dealId1"},
			{mid: 32345, code: "code2"}, // no bid for ad unit
			{mid: 32346, code: "code3", price: 1.24, content: "banner-content2", dealId: "dealId2"},
		},
		secure: true,
	}
	return testData
}

func createOpenRtbRequest(testData *aBidInfo) *openrtb.BidRequest {
	secure := int8(0)
	if testData.secure {
		secure = int8(1)
	}
	bidRequest := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{
			{
				ID:     testData.tags[0].code,
				Secure: &secure,
				Ext:    openrtb.RawJSON(`{"bidder": { "mid": "32344" }}`),
				Banner: &openrtb.Banner{},
			},
			{
				ID:     testData.tags[1].code,
				Secure: &secure,
				Ext:    openrtb.RawJSON(`{"bidder": { "mid": 32345 }}`),
				Banner: &openrtb.Banner{},
			},
			{
				ID:     testData.tags[2].code,
				Secure: &secure,
				Ext:    openrtb.RawJSON(`{"bidder": { "mid": 32346 }}`),
				Banner: &openrtb.Banner{},
			},
		},
		Site: &openrtb.Site{
			Page: testData.referrer,
		},
		Device: &openrtb.Device{
			UA: testData.deviceUA,
			IP: testData.deviceIP,
		},
		Source: &openrtb.Source{
			TID: testData.tid,
		},
		User: &openrtb.User{
			BuyerUID: testData.buyerUID,
		},
	}
	return bidRequest
}

func TestOpenRTBStandardResponse(t *testing.T) {
	testData := createTestData()
	request := createOpenRtbRequest(testData)

	responseBody, err := createAdformServerResponse(*testData)
	if err != nil {
		t.Fatalf("Unable to create server response: %v", err)
		return
	}
	httpResponse := &adapters.ResponseData{StatusCode: http.StatusOK, Body: responseBody}

	bidder := new(AdformAdapter)
	bids, errs := bidder.MakeBids(request, nil, httpResponse)

	if len(bids) != 2 {
		t.Fatalf("Expected 2 bids. Got %d", len(bids))
	}
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors. Got %d", len(errs))
	}

	for _, typeBid := range bids {
		if typeBid.BidType != openrtb_ext.BidTypeBanner {
			t.Errorf("Expected a banner bid. Got: %s", bids[0].BidType)
		}
		bid := typeBid.Bid
		matched := false

		for _, tag := range testData.tags {
			if bid.ID == tag.code {
				matched = true
				if bid.Price != tag.price {
					t.Errorf("Incorrect bid price '%.2f' expected '%.2f'", bid.Price, tag.price)
				}
				if bid.W != testData.width || bid.H != testData.height {
					t.Errorf("Incorrect bid size %dx%d, expected %dx%d", bid.W, bid.H, testData.width, testData.height)
				}
				if bid.AdM != tag.content {
					t.Errorf("Incorrect bid markup '%s' expected '%s'", bid.AdM, tag.content)
				}
				if bid.DealID != tag.dealId {
					t.Errorf("Incorrect deal id '%s' expected '%s'", bid.DealID, tag.dealId)
				}
			}
		}
		if !matched {
			t.Errorf("Received bid with unknown id '%s'", bid.ID)
		}
	}
}

func TestOpenRTBSurpriseResponse(t *testing.T) {
	bidder := new(AdformAdapter)

	bids, errs := bidder.MakeBids(nil, nil,
		&adapters.ResponseData{StatusCode: http.StatusNoContent, Body: []byte("")})
	if bids != nil && errs != nil {
		t.Fatalf("Expected no bids and no errors. Got %d bids and %d", len(bids), len(errs))
	}

	bids, errs = bidder.MakeBids(nil, nil,
		&adapters.ResponseData{StatusCode: http.StatusServiceUnavailable, Body: []byte("")})
	if bids != nil || len(errs) != 1 {
		t.Fatalf("Expected one error and no bids. Got %d bids and %d", len(bids), len(errs))
	}

	bids, errs = bidder.MakeBids(nil, nil,
		&adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte("{:'not-valid-json'}")})
	if bids != nil || len(errs) != 1 {
		t.Fatalf("Expected one error and no bids. Got %d bids and %d", len(bids), len(errs))
	}
}

// Properties tests

func TestAdformProperties(t *testing.T) {
	adapter := NewAdformAdapter(adapters.DefaultHTTPAdapterConfig, "adx.adform.net/adx")

	if adapter.SkipNoCookies() != false {
		t.Fatalf("should have been false")
	}
	if adapter.Name() != "Adform" {
		t.Fatalf("should have been Adform")
	}
}

// helpers

func assertAdformServerRequest(testData aBidInfo, r *http.Request) *string {
	if ok, err := equal("GET", r.Method, "HTTP method"); !ok {
		return err
	}
	if testData.secure {
		if ok, err := equal("https", r.URL.Scheme, "Scheme"); !ok {
			return err
		}
	}
	if ok, err := equal("CC=1&rp=4&fd=1&stid=transaction-id&bWlkPTMyMzQ0&bWlkPTMyMzQ1&bWlkPTMyMzQ2", r.URL.RawQuery, "Query string"); !ok {
		return err
	}
	if ok, err := equal("application/json;charset=utf-8", r.Header.Get("Content-Type"), "Content type"); !ok {
		return err
	}
	if ok, err := equal(testData.deviceUA, r.Header.Get("User-Agent"), "User agent"); !ok {
		return err
	}
	if ok, err := equal(testData.deviceIP, r.Header.Get("X-Forwarded-For"), "Device IP"); !ok {
		return err
	}
	if ok, err := equal(testData.referrer, r.Header.Get("Referer"), "Referer"); !ok {
		return err
	}
	if ok, err := equal(fmt.Sprintf("uid=%s", testData.buyerUID), r.Header.Get("Cookie"), "Buyer ID"); !ok {
		return err
	}
	return nil
}

func equal(expected string, actual string, message string) (bool, *string) {
	if expected != actual {
		message := fmt.Sprintf("%s '%s' doesn't match '%s'", message, actual, expected)
		return false, &message
	}
	return true, nil
}
