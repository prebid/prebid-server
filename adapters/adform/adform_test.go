package adform

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/usersync"

	"fmt"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "adformtest", NewAdformBidder(nil, "http://adx.adform.net/adx"))
}

type aTagInfo struct {
	mid       uint32
	priceType string
	keyValues string
	keyWords  string
	code      string

	price      float64
	content    string
	dealId     string
	creativeId string
}

type aBidInfo struct {
	deviceIP  string
	deviceUA  string
	deviceIFA string
	tags      []aTagInfo
	referrer  string
	width     uint64
	height    uint64
	tid       string
	buyerUID  string
	secure    bool
	currency  string
	delay     time.Duration
}

var adformTestData aBidInfo

// Legacy auction tests

func DummyAdformServer(w http.ResponseWriter, r *http.Request) {
	errorString := assertAdformServerRequest(adformTestData, r, false)
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
			Currency:     "EUR",
			Width:        testData.width,
			Height:       testData.height,
			DealId:       testData.tags[0].dealId,
			CreativeId:   testData.tags[0].creativeId,
		},
		{},
		{
			ResponseType: "banner",
			Banner:       testData.tags[2].content,
			Price:        testData.tags[2].price,
			Currency:     "EUR",
			Width:        testData.width,
			Height:       testData.height,
			DealId:       testData.tags[2].dealId,
			CreativeId:   testData.tags[2].creativeId,
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
				if bid.Creative_id != tag.creativeId {
					t.Errorf("Incorrect creative id '%s' expected '%s'", bid.Creative_id, tag.creativeId)
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
	adformTestData = createTestData(false)

	// prepare adapter
	conf := *adapters.DefaultHTTPAdapterConfig
	adapter := NewAdformAdapter(&conf, server.URL)

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

	pbsCookie := usersync.ParsePBSCookieFromRequest(prebidHttpRequest, &config.HostCookie{})
	pbsCookie.TrySync("adform", adformTestData.buyerUID)
	fakeWriter := httptest.NewRecorder()

	pbsCookie.SetCookieOnResponse(fakeWriter, false, &config.HostCookie{Domain: ""}, time.Minute)
	prebidHttpRequest.Header.Add("Cookie", fakeWriter.Header().Get("Set-Cookie"))

	cacheClient, _ := dummycache.New()
	r, err := pbs.ParsePBSRequest(prebidHttpRequest, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, cacheClient, &config.HostCookie{})
	if err != nil {
		t.Fatalf("ParsePBSRequest failed: %v", err)
	}
	if len(r.Bidders) != 1 {
		t.Fatalf("ParsePBSRequest returned %d bidders instead of 1", len(r.Bidders))
	}
	if r.Bidders[0].BidderCode != "adform" {
		t.Fatalf("ParsePBSRequest returned invalid bidder")
	}

	// can't be set in preparePrebidRequestBody as will be lost during json serialization and deserialization
	// for the adapters which don't support OpenRTB requests the old PBSRequest is created from OpenRTB request
	// so User and Regs are copied from OpenRTB request, see legacy.go -> toLegacyRequest
	regs := getRegs()
	r.Regs = &regs
	user := openrtb.User{
		Ext: getUserExt(),
	}
	r.User = &user

	return r
}

func preparePrebidRequestBody(requestData aBidInfo, t *testing.T) *bytes.Buffer {
	prebidRequest := pbs.PBSRequest{
		AdUnits: make([]pbs.AdUnit, 3),
		Device: &openrtb.Device{
			UA:  requestData.deviceUA,
			IP:  requestData.deviceIP,
			IFA: requestData.deviceIFA,
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
					Params:     json.RawMessage(formatAdUnitJson(tag)),
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
	bidder := NewAdformBidder(nil, "http://adx.adform.net")

	testData := createTestData(true)
	request := createOpenRtbRequest(&testData)

	httpRequests, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

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

	errorString := assertAdformServerRequest(testData, r, true)
	if errorString != nil {
		t.Errorf("Request error: %s", *errorString)
	}
}

func TestOpenRTBIncorrectRequest(t *testing.T) {
	bidder := new(AdformAdapter)
	request := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{
			{ID: "incorrect-bidder-field", Ext: json.RawMessage(`{"bidder1": { "mid": "32344" }}`)},
			{ID: "incorrect-adform-params", Ext: json.RawMessage(`{"bidder": { : "33" }}`)},
			{ID: "mid-integer", Ext: json.RawMessage(`{"bidder": { "mid": 1.234 }}`)},
			{ID: "mid-greater-then-zero", Ext: json.RawMessage(`{"bidder": { "mid": -1 }}`)},
		},
		Device: &openrtb.Device{UA: "ua", IP: "ip"},
		User:   &openrtb.User{BuyerUID: "buyerUID"},
	}

	httpRequests, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	if len(errs) != len(request.Imp) {
		t.Errorf("%d Imp objects should have errors. but was %d errors", len(request.Imp), len(errs))
	}
	if len(httpRequests) != 0 {
		t.Fatalf("All Imp objects have errors, but requests count: %d. Expected %d", len(httpRequests), 0)
	}
}

func createTestData(secure bool) aBidInfo {
	testData := aBidInfo{
		deviceIP:  "111.111.111.111",
		deviceUA:  "Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_1 like Mac OS X) AppleWebKit/603.1.30 (KHTML, like Gecko) Mobile/14E8301",
		deviceIFA: "6D92078A-8246-4BA4-AE5B-76104861E7DC",
		referrer:  "http://test.com",
		tid:       "transaction-id",
		buyerUID:  "user-id",
		tags: []aTagInfo{
			{mid: 32344, keyValues: "color:red,age:30-40", keyWords: "red,blue", priceType: "gross", code: "code1", price: 1.23, content: "banner-content1", dealId: "dealId1", creativeId: "creativeId1"},
			{mid: 32345, priceType: "net", code: "code2"}, // no bid for ad unit
			{mid: 32346, code: "code3", price: 1.24, content: "banner-content2", dealId: "dealId2"},
		},
		secure:   secure,
		currency: "EUR",
	}
	return testData
}

func createOpenRtbRequest(testData *aBidInfo) *openrtb.BidRequest {
	secure := int8(0)
	if testData.secure {
		secure = int8(1)
	}

	bidRequest := &openrtb.BidRequest{
		ID:  "test-request-id",
		Imp: make([]openrtb.Imp, len(testData.tags)),
		Site: &openrtb.Site{
			Page: testData.referrer,
		},
		Device: &openrtb.Device{
			UA:  testData.deviceUA,
			IP:  testData.deviceIP,
			IFA: testData.deviceIFA,
		},
		Source: &openrtb.Source{
			TID: testData.tid,
		},
		User: &openrtb.User{
			BuyerUID: testData.buyerUID,
		},
	}
	for i, tag := range testData.tags {
		bidRequest.Imp[i] = openrtb.Imp{
			ID:     tag.code,
			Secure: &secure,
			Ext:    json.RawMessage(fmt.Sprintf("{\"bidder\": %s}", formatAdUnitJson(tag))),
			Banner: &openrtb.Banner{},
		}
	}

	regs := getRegs()
	bidRequest.Regs = &regs
	bidRequest.User.Ext = getUserExt()

	bidRequest.Cur = make([]string, 1)
	bidRequest.Cur[0] = testData.currency

	return bidRequest
}

func TestOpenRTBStandardResponse(t *testing.T) {
	testData := createTestData(true)
	request := createOpenRtbRequest(&testData)

	responseBody, err := createAdformServerResponse(testData)
	if err != nil {
		t.Fatalf("Unable to create server response: %v", err)
		return
	}
	httpResponse := &adapters.ResponseData{StatusCode: http.StatusOK, Body: responseBody}

	bidder := new(AdformAdapter)
	bidResponse, errs := bidder.MakeBids(request, nil, httpResponse)

	if len(bidResponse.Bids) != 2 {
		t.Fatalf("Expected 2 bids. Got %d", len(bidResponse.Bids))
	}
	if len(errs) != 0 {
		t.Errorf("Expected 0 errors. Got %d", len(errs))
	}

	for _, typeBid := range bidResponse.Bids {
		if typeBid.BidType != openrtb_ext.BidTypeBanner {
			t.Errorf("Expected a banner bid. Got: %s", bidResponse.Bids[0].BidType)
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
				if bid.CrID != tag.creativeId {
					t.Errorf("Incorrect creative id '%s' expected '%s'", bid.CrID, tag.creativeId)
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

	bidResponse, errs := bidder.MakeBids(nil, nil,
		&adapters.ResponseData{StatusCode: http.StatusNoContent, Body: []byte("")})
	if bidResponse != nil && errs != nil {
		t.Fatalf("Expected no bids and no errors. Got %d bids and %d", len(bidResponse.Bids), len(errs))
	}

	bidResponse, errs = bidder.MakeBids(nil, nil,
		&adapters.ResponseData{StatusCode: http.StatusServiceUnavailable, Body: []byte("")})
	if bidResponse != nil || len(errs) != 1 {
		t.Fatalf("Expected one error and no bids. Got %d bids and %d", len(bidResponse.Bids), len(errs))
	}

	bidResponse, errs = bidder.MakeBids(nil, nil,
		&adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte("{:'not-valid-json'}")})
	if bidResponse != nil || len(errs) != 1 {
		t.Fatalf("Expected one error and no bids. Got %d bids and %d", len(bidResponse.Bids), len(errs))
	}
}

// Properties tests

func TestAdformProperties(t *testing.T) {
	adapter := NewAdformAdapter(adapters.DefaultHTTPAdapterConfig, "adx.adform.net/adx")

	if adapter.SkipNoCookies() != false {
		t.Fatalf("should have been false")
	}
}

// helpers

func getRegs() openrtb.Regs {
	var gdpr int8 = 1
	regsExt := openrtb_ext.ExtRegs{
		GDPR: &gdpr,
	}
	regs := openrtb.Regs{}
	regsExtData, err := json.Marshal(regsExt)
	if err == nil {
		regs.Ext = regsExtData
	}
	return regs
}

func getUserExt() []byte {
	digitrust := openrtb_ext.ExtUserDigiTrust{
		ID:   "digitrustId",
		KeyV: 1,
		Pref: 0,
	}

	eids := []openrtb_ext.ExtUserEid{
		{
			Source: "test.com",
			Uids: []openrtb_ext.ExtUserEidUid{
				{
					ID:    "some_user_id",
					Atype: 1,
				},
				{
					ID: "other_user_id",
				},
			},
		},
		{
			Source: "test2.org",
			Uids: []openrtb_ext.ExtUserEidUid{
				{
					ID:    "other_user_id",
					Atype: 2,
				},
			},
		},
	}

	userExt := openrtb_ext.ExtUser{
		Eids:      eids,
		Consent:   "abc",
		DigiTrust: &digitrust,
	}
	userExtData, err := json.Marshal(userExt)
	if err == nil {
		return userExtData
	}

	return nil
}

func formatAdUnitJson(tag aTagInfo) string {
	return fmt.Sprintf("{ \"mid\": %d%s%s%s}",
		tag.mid,
		formatAdUnitParam("priceType", tag.priceType),
		formatAdUnitParam("mkv", tag.keyValues),
		formatAdUnitParam("mkw", tag.keyWords))
}

func formatAdUnitParam(fieldName string, fieldValue string) string {
	if fieldValue != "" {
		return fmt.Sprintf(", \"%s\": \"%s\"", fieldName, fieldValue)
	}

	return ""
}

func assertAdformServerRequest(testData aBidInfo, r *http.Request, isOpenRtb bool) *string {
	if ok, err := equal("GET", r.Method, "HTTP method"); !ok {
		return err
	}
	if testData.secure {
		if ok, err := equal("https", r.URL.Scheme, "Scheme"); !ok {
			return err
		}
	}

	var midsWithCurrency = ""
	var queryString = ""
	if isOpenRtb {
		midsWithCurrency = "bWlkPTMyMzQ0JnJjdXI9RVVSJm1rdj1jb2xvcjpyZWQsYWdlOjMwLTQwJm1rdz1yZWQsYmx1ZQ&bWlkPTMyMzQ1JnJjdXI9RVVS&bWlkPTMyMzQ2JnJjdXI9RVVS"
		queryString = "CC=1&adid=6D92078A-8246-4BA4-AE5B-76104861E7DC&eids=eyJ0ZXN0LmNvbSI6eyJvdGhlcl91c2VyX2lkIjpbMF0sInNvbWVfdXNlcl9pZCI6WzFdfSwidGVzdDIub3JnIjp7Im90aGVyX3VzZXJfaWQiOlsyXX19&fd=1&gdpr=1&gdpr_consent=abc&ip=111.111.111.111&pt=gross&rp=4&stid=transaction-id&" + midsWithCurrency
	} else {
		midsWithCurrency = "bWlkPTMyMzQ0JnJjdXI9VVNEJm1rdj1jb2xvcjpyZWQsYWdlOjMwLTQwJm1rdz1yZWQsYmx1ZQ&bWlkPTMyMzQ1JnJjdXI9VVNE&bWlkPTMyMzQ2JnJjdXI9VVNE" // no way to pass currency in legacy adapter
		queryString = "CC=1&adid=6D92078A-8246-4BA4-AE5B-76104861E7DC&fd=1&gdpr=1&gdpr_consent=abc&ip=111.111.111.111&pt=gross&rp=4&stid=transaction-id&" + midsWithCurrency
	}

	if ok, err := equal(queryString, r.URL.RawQuery, "Query string"); !ok {
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
	if ok, err := equal(fmt.Sprintf("uid=%s;DigiTrust.v1.identity=eyJpZCI6ImRpZ2l0cnVzdElkIiwidmVyc2lvbiI6MSwia2V5diI6MSwicHJpdmFjeSI6eyJvcHRvdXQiOmZhbHNlfX0", testData.buyerUID), r.Header.Get("Cookie"), "Buyer ID"); !ok {
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

// Price type parameter tests

func TestPriceTypeValidation(t *testing.T) {
	// Arrange
	priceTypeTestCases := map[string]bool{
		"net":   true,
		"NET":   true,
		"nEt":   true,
		"nt":    false,
		"gross": true,
		"GROSS": true,
		"groSS": true,
		"gorss": false,
		"":      false,
	}

	// Act
	for priceType, expected := range priceTypeTestCases {
		_, valid := isPriceTypeValid(priceType)

		// Assert
		if expected != valid {
			t.Fatalf("Unexpected result for '%s' price type. Got valid = %s. Expected valid = %s", priceType, strconv.FormatBool(valid), strconv.FormatBool(expected))
		}
	}
}

func TestPriceTypeUrlParameterCreation(t *testing.T) {
	// Arrange
	priceTypeParameterTestCases := map[string][]*adformAdUnit{
		"":      {{MasterTagId: "123"}, {MasterTagId: "456"}},
		"net":   {{MasterTagId: "123", PriceType: priceTypeNet}, {MasterTagId: "456"}, {MasterTagId: "789", PriceType: priceTypeNet}},
		"gross": {{MasterTagId: "123", PriceType: priceTypeNet}, {MasterTagId: "456", PriceType: priceTypeGross}, {MasterTagId: "789", PriceType: priceTypeNet}},
	}

	// Act
	for expected, adUnits := range priceTypeParameterTestCases {
		parameter := getValidPriceTypeParameter(adUnits)

		// Assert
		if expected != parameter {
			t.Fatalf("Unexpected result for price type parameter. Got '%s'. Expected '%s'", parameter, expected)
		}
	}
}

// Asserts that toOpenRtbBidResponse() creates a *adapters.BidderResponse with
// the currency of the last valid []*adformBid element and the expected number of bids
func TestToOpenRtbBidResponse(t *testing.T) {
	expectedBids := 3
	lastCurrency, anotherCurrency, emptyCurrency := "EUR", "USD", ""

	request := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{
			{
				ID:     "banner-imp-no1",
				Ext:    json.RawMessage(`{"bidder1": { "mid": "32341" }}`),
				Banner: &openrtb.Banner{},
			},
			{
				ID:     "banner-imp-no2",
				Ext:    json.RawMessage(`{"bidder1": { "mid": "32342" }}`),
				Banner: &openrtb.Banner{},
			},
			{
				ID:     "banner-imp-no3",
				Ext:    json.RawMessage(`{"bidder1": { "mid": "32343" }}`),
				Banner: &openrtb.Banner{},
			},
			{
				ID:     "banner-imp-no4",
				Ext:    json.RawMessage(`{"bidder1": { "mid": "32344" }}`),
				Banner: &openrtb.Banner{},
			},
		},
		Device: &openrtb.Device{UA: "ua", IP: "ip"},
		User:   &openrtb.User{BuyerUID: "buyerUID"},
	}

	testAdformBids := []*adformBid{
		{
			ResponseType: "banner",
			Banner:       "banner-content1",
			Price:        1.23,
			Currency:     anotherCurrency,
			Width:        300,
			Height:       200,
			DealId:       "dealId1",
			CreativeId:   "creativeId1",
		},
		{},
		{
			ResponseType: "banner",
			Banner:       "banner-content3",
			Price:        1.24,
			Currency:     emptyCurrency,
			Width:        300,
			Height:       200,
			DealId:       "dealId3",
			CreativeId:   "creativeId3",
		},
		{
			ResponseType: "banner",
			Banner:       "banner-content4",
			Price:        1.25,
			Currency:     lastCurrency,
			Width:        300,
			Height:       200,
			DealId:       "dealId4",
			CreativeId:   "creativeId4",
		},
	}

	actualBidResponse := toOpenRtbBidResponse(testAdformBids, request)

	assert.Equalf(t, expectedBids, len(actualBidResponse.Bids), "bid count")
	assert.Equalf(t, lastCurrency, actualBidResponse.Currency, "currency")
}
