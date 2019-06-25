package openrtb2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-server/stored_requests"
	metrics "github.com/rcrowley/go-metrics"

	"github.com/buger/jsonparser"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/mxmCherry/openrtb"
	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/stretchr/testify/assert"
)

const maxSize = 1024 * 256

// Struct of data for the general purpose auction tester
type getResponseFromDirectory struct {
	dir             string
	payloadGetter   func(*testing.T, []byte) []byte
	messageGetter   func(*testing.T, []byte) []byte
	expectedCode    int
	aliased         bool
	disabledBidders []string
}

// TestExplicitUserId makes sure that the cookie's ID doesn't override an explicit value sent in the request.
func TestExplicitUserId(t *testing.T) {
	cookieName := "userid"
	mockId := "12345"
	cfg := &config.Configuration{
		MaxRequestSize: maxSize,
		HostCookie: config.HostCookie{
			CookieName: cookieName,
		},
	}
	ex := &mockExchange{}

	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(`{
		"id": "some-request-id",
		"site": {
			"page": "test.somepage.com"
		},
		"user": {
			"id": "explicit"
		},
		"imp": [
			{
				"id": "my-imp-id",
				"banner": {
					"format": [
						{
							"w": 300,
							"h": 600
						}
					]
				},
				"pmp": {
					"deals": [
						{
							"id": "some-deal-id"
						}
					]
				},
				"ext": {
					"appnexus": {
						"placementId": 10433394
					}
				}
			}
		]
	}`))
	request.AddCookie(&http.Cookie{
		Name:  cookieName,
		Value: mockId,
	})
	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	endpoint, _ := NewEndpoint(ex, newParamsValidator(t), empty_fetcher.EmptyFetcher{}, empty_fetcher.EmptyFetcher{}, cfg, theMetrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), map[string]string{}, []byte{}, openrtb_ext.BidderMap)

	endpoint(httptest.NewRecorder(), request, nil)

	if ex.lastRequest == nil {
		t.Fatalf("The request never made it into the Exchange.")
	}

	if ex.lastRequest.User == nil {
		t.Fatalf("The exchange should have received a request with a non-nil user.")
	}

	if ex.lastRequest.User.ID != "explicit" {
		t.Errorf("Bad User ID. Expected explicit, got %s", ex.lastRequest.User.ID)
	}
}

// TestImplicitUserId makes sure that that bidrequest.user.id gets populated from the host cookie, if it wasn't sent explicitly.
func TestImplicitUserId(t *testing.T) {
	cookieName := "userid"
	mockId := "12345"
	cfg := &config.Configuration{
		MaxRequestSize: maxSize,
		HostCookie: config.HostCookie{
			CookieName: cookieName,
		},
	}
	ex := &mockExchange{}

	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	request.AddCookie(&http.Cookie{
		Name:  cookieName,
		Value: mockId,
	})
	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())

	endpoint, _ := NewEndpoint(ex, newParamsValidator(t), empty_fetcher.EmptyFetcher{}, empty_fetcher.EmptyFetcher{}, cfg, theMetrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), map[string]string{}, []byte{}, openrtb_ext.BidderMap)
	endpoint(httptest.NewRecorder(), request, nil)

	if ex.lastRequest == nil {
		t.Fatalf("The request never made it into the Exchange.")
	}

	if ex.lastRequest.User == nil {
		t.Fatalf("The exchange should have received a request with a non-nil user.")
	}

	if ex.lastRequest.User.ID != mockId {
		t.Errorf("Bad User ID. Expected %s, got %s", mockId, ex.lastRequest.User.ID)
	}
}

// TestGoodRequests makes sure we return 200s on good requests.
func TestGoodRequests(t *testing.T) {
	exemplary := &getResponseFromDirectory{
		dir:           "sample-requests/valid-whole/exemplary",
		payloadGetter: getRequestPayload,
		messageGetter: nilReturner,
		expectedCode:  http.StatusOK,
		aliased:       true,
	}
	supplementary := &getResponseFromDirectory{
		dir:           "sample-requests/valid-whole/supplementary",
		payloadGetter: noop,
		messageGetter: nilReturner,
		expectedCode:  http.StatusOK,
		aliased:       true,
	}
	exemplary.assert(t)
	supplementary.assert(t)
}

// TestGoodNativeRequests makes sure we return 200s on well-formed Native requests.
func TestGoodNativeRequests(t *testing.T) {
	tests := &getResponseFromDirectory{
		dir:           "sample-requests/valid-native",
		payloadGetter: buildNativeRequest,
		messageGetter: nilReturner,
		expectedCode:  http.StatusOK,
		aliased:       true,
	}
	tests.assert(t)
}

// TestBadRequests makes sure we return 400s on bad requests.
func TestBadRequests(t *testing.T) {
	// Need to turn off aliases for bad requests as applying the alias can fail on a bad request before the expected error is reached.
	tests := &getResponseFromDirectory{
		dir:           "sample-requests/invalid-whole",
		payloadGetter: getRequestPayload,
		messageGetter: getMessage,
		expectedCode:  http.StatusBadRequest,
		aliased:       false,
	}
	tests.assert(t)
}

// TestBadRequests makes sure we return 400s on requests with bad Native requests.
func TestBadNativeRequests(t *testing.T) {
	tests := &getResponseFromDirectory{
		dir:           "sample-requests/invalid-native",
		payloadGetter: buildNativeRequest,
		messageGetter: nilReturner,
		expectedCode:  http.StatusBadRequest,
		aliased:       false,
	}
	tests.assert(t)
}

// TestAliasedRequests makes sure we handle (defuault) aliased bidders properly
func TestAliasedRequests(t *testing.T) {
	tests := &getResponseFromDirectory{
		dir:           "sample-requests/aliased",
		payloadGetter: noop,
		messageGetter: nilReturner,
		expectedCode:  http.StatusOK,
		aliased:       true,
	}
	tests.assert(t)
}

// TestDisabledBidders makes sure we don't break when encountering a disabled bidder
func TestDisabledBidders(t *testing.T) {
	badTests := &getResponseFromDirectory{
		dir:             "sample-requests/disabled/bad",
		payloadGetter:   getRequestPayload,
		messageGetter:   getMessage,
		expectedCode:    http.StatusBadRequest,
		aliased:         false,
		disabledBidders: []string{"appnexus", "rubicon"},
	}
	goodTests := &getResponseFromDirectory{
		dir:             "sample-requests/disabled/good",
		payloadGetter:   noop,
		messageGetter:   nilReturner,
		expectedCode:    http.StatusOK,
		aliased:         false,
		disabledBidders: []string{"appnexus", "rubicon"},
	}
	badTests.assert(t)
	goodTests.assert(t)
}

// assertResponseFromDirectory makes sure that the payload from each file in dir gets the expected response status code
// from the /openrtb2/auction endpoint.
func (gr *getResponseFromDirectory) assert(t *testing.T) {
	//t *testing.T, dir string, payloadGetter func(*testing.T, []byte) []byte, messageGetter func(*testing.T, []byte) []byte, expectedCode int, aliased bool) {
	t.Helper()
	for _, fileInfo := range fetchFiles(t, gr.dir) {
		filename := gr.dir + "/" + fileInfo.Name()
		fileData := readFile(t, filename)
		code, msg := gr.doRequest(t, gr.payloadGetter(t, fileData))
		assertResponseCode(t, filename, code, gr.expectedCode, msg)

		expectMsg := gr.messageGetter(t, fileData)
		if len(expectMsg) > 0 {
			assert.Equal(t, string(expectMsg), msg, "file %s had bad response body", filename)
		}
	}
}

// fetchFiles returns a list of the files from dir, or fails the test if an error occurs.
func fetchFiles(t *testing.T, dir string) []os.FileInfo {
	t.Helper()
	requestFiles, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("Failed to read folder: %s", dir)
	}
	return requestFiles
}

func readFile(t *testing.T, filename string) []byte {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", filename, err)
	}
	return data
}

// doRequest populates the app with mock dependencies and sends requestData to the /openrtb2/auction endpoint.
func (gr *getResponseFromDirectory) doRequest(t *testing.T, requestData []byte) (int, string) {
	aliasJSON := []byte{}
	if gr.aliased {
		aliasJSON = []byte(`{"ext":{"prebid":{"aliases": {"test1": "appnexus", "test2": "rubicon", "test3": "openx"}}}}`)
	}
	disabledBidders := map[string]string{
		"indexExchange": "Bidder \"indexExchange\" has been deprecated and is no longer available. Please use bidder \"ix\" and note that the bidder params have changed.",
	}
	adapterCfg := blankAdapterConfig(openrtb_ext.BidderList(), gr.disabledBidders)
	_, bidderMap := exchange.DisableBidders(adapterCfg, openrtb_ext.BidderList(), disabledBidders)

	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	endpoint, _ := NewEndpoint(&nobidExchange{}, newParamsValidator(t), &mockStoredReqFetcher{}, empty_fetcher.EmptyFetcher{}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), disabledBidders, aliasJSON, bidderMap)

	request := httptest.NewRequest("POST", "/openrtb2/auction", bytes.NewReader(requestData))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)
	return recorder.Code, recorder.Body.String()
}

// TestBadAliasRequests() reuses two requests that would fail anyway.  Here, we
// take advantage of our knowledge that processStoredRequests() in auction.go
// processes aliases before it processes stored imps.  Changing that order
// would probably cause this test to fail.
func TestBadAliasRequests(t *testing.T) {
	doBadAliasRequest(t, "sample-requests/invalid-stored/bad_stored_imp.json", "Invalid request: Invalid JSON in Default Request Settings: invalid character '\"' after object key:value pair at offset 51\n")
	doBadAliasRequest(t, "sample-requests/invalid-stored/bad_incoming_imp.json", "Invalid request: Invalid JSON in Incoming Request: invalid character '\"' after object key:value pair at offset 230\n")
}

// doBadAliasRequest() is a customized variation of doRequest(), above
func doBadAliasRequest(t *testing.T, filename string, expectMsg string) {
	t.Helper()
	fileData := readFile(t, filename)
	requestData := getRequestPayload(t, fileData)
	// aliasJSON lacks a comma after the "appnexus" entry so is bad JSON
	aliasJSON := []byte(`{"ext":{"prebid":{"aliases": {"test1": "appnexus" "test2": "rubicon", "test3": "openx"}}}}`)
	disabledBidders := map[string]string{
		"indexExchange": "Bidder \"indexExchange\" has been deprecated and is no longer available. Please use bidder \"ix\" and note that the bidder params have changed.",
	}
	adapterCfg := blankAdapterConfig(openrtb_ext.BidderList(), []string{""})
	_, bidderMap := exchange.DisableBidders(adapterCfg, openrtb_ext.BidderList(), disabledBidders)

	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	endpoint, _ := NewEndpoint(&nobidExchange{}, newParamsValidator(t), &mockStoredReqFetcher{}, empty_fetcher.EmptyFetcher{}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), disabledBidders, aliasJSON, bidderMap)

	request := httptest.NewRequest("POST", "/openrtb2/auction", bytes.NewReader(requestData))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	assertResponseCode(t, filename, recorder.Code, http.StatusBadRequest, recorder.Body.String())
	assert.Equal(t, string(expectMsg), recorder.Body.String(), "file %s had bad response body", filename)

}

func newParamsValidator(t *testing.T) openrtb_ext.BidderParamValidator {
	paramValidator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Error creating the param validator: %v", err)
	}
	return paramValidator
}

func assertResponseCode(t *testing.T, filename string, actual int, expected int, msg string) {
	t.Helper()
	if actual != expected {
		t.Errorf("Expected a %d response from %v. Got %d: %s", expected, filename, actual, msg)
	}
}

// buildNativeRequest JSON-encodes the nativeData as a string, and puts it into request.imp[0].native.request
// of a request which is valid otherwise.
func buildNativeRequest(t *testing.T, nativeData []byte) []byte {
	serialized, err := json.Marshal(string(nativeData))
	if err != nil {
		t.Fatalf("Failed to string-escape JSON data: %v", err)
	}

	buf := bytes.NewBuffer(nil)
	buf.WriteString(`{"id":"req-id","site":{"page":"some.page.com"},"tmax":500,"imp":[{"id":"some-imp","native":{"request":`)
	buf.Write(serialized)
	buf.WriteString(`},"ext":{"appnexus":{"placementId":10433394}}}]}`)
	return buf.Bytes()
}

func noop(t *testing.T, data []byte) []byte {
	return data
}

func nilReturner(t *testing.T, data []byte) []byte {
	return nil
}

func getRequestPayload(t *testing.T, example []byte) []byte {
	t.Helper()
	if value, _, _, err := jsonparser.Get(example, "requestPayload"); err != nil {
		t.Fatalf("Error parsing root.requestPayload from request: %v.", err)
	} else {
		return value
	}
	return nil
}

// TestNilExchange makes sure we fail when given nil for the Exchange.
func TestNilExchange(t *testing.T) {
	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	_, err := NewEndpoint(nil, newParamsValidator(t), empty_fetcher.EmptyFetcher{}, empty_fetcher.EmptyFetcher{}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), map[string]string{}, []byte{}, openrtb_ext.BidderMap)
	if err == nil {
		t.Errorf("NewEndpoint should return an error when given a nil Exchange.")
	}
}

// TestNilValidator makes sure we fail when given nil for the BidderParamValidator.
func TestNilValidator(t *testing.T) {
	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	_, err := NewEndpoint(&nobidExchange{}, nil, empty_fetcher.EmptyFetcher{}, empty_fetcher.EmptyFetcher{}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), map[string]string{}, []byte{}, openrtb_ext.BidderMap)
	if err == nil {
		t.Errorf("NewEndpoint should return an error when given a nil BidderParamValidator.")
	}
}

// TestExchangeError makes sure we return a 500 if the exchange auction fails.
func TestExchangeError(t *testing.T) {
	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	endpoint, _ := NewEndpoint(&brokenExchange{}, newParamsValidator(t), empty_fetcher.EmptyFetcher{}, empty_fetcher.EmptyFetcher{}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), map[string]string{}, []byte{}, openrtb_ext.BidderMap)
	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d. Got %d. Input was: %s", http.StatusInternalServerError, recorder.Code, validRequest(t, "site.json"))
	}
}

// TestUserAgentSetting makes sure we read the User-Agent header if it wasn't defined on the request.
func TestUserAgentSetting(t *testing.T) {
	httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	httpReq.Header.Set("User-Agent", "foo")
	bidReq := &openrtb.BidRequest{}

	setUAImplicitly(httpReq, bidReq)

	if bidReq.Device == nil {
		t.Fatal("bidrequest.device should have been set implicitly.")
	}
	if bidReq.Device.UA != "foo" {
		t.Errorf("bidrequest.device.ua should have been \"foo\". Got %s", bidReq.Device.UA)
	}
}

// TestUserAgentOverride makes sure that the explicit UA from the request takes precedence.
func TestUserAgentOverride(t *testing.T) {
	httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	httpReq.Header.Set("User-Agent", "foo")
	bidReq := &openrtb.BidRequest{
		Device: &openrtb.Device{
			UA: "bar",
		},
	}

	setUAImplicitly(httpReq, bidReq)

	if bidReq.Device.UA != "bar" {
		t.Errorf("bidrequest.device.ua should have been \"bar\". Got %s", bidReq.Device.UA)
	}
}

func TestAuctionTypeDefault(t *testing.T) {
	bidReq := &openrtb.BidRequest{}
	setAuctionTypeImplicitly(bidReq)

	if bidReq.AT != 1 {
		t.Errorf("Expected request.at to be 1. Got %d", bidReq.AT)
	}
}

// TestImplicitIPs prevents #230
func TestImplicitIPs(t *testing.T) {
	ex := &nobidExchange{}
	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	endpoint, _ := NewEndpoint(ex, newParamsValidator(t), &mockStoredReqFetcher{}, empty_fetcher.EmptyFetcher{}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), map[string]string{}, []byte{}, openrtb_ext.BidderMap)

	httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	httpReq.Header.Set("X-Forwarded-For", "123.456.78.90")
	recorder := httptest.NewRecorder()

	endpoint(recorder, httpReq, nil)

	if ex.gotRequest == nil {
		t.Fatalf("The request never made it into the Exchange.")
	}

	if ex.gotRequest.Device.IP != "123.456.78.90" {
		t.Errorf("Bad device IP. Expected 123.456.78.90, got %s", ex.gotRequest.Device.IP)
	}
}

func TestImplicitSecure(t *testing.T) {
	httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	httpReq.Header.Set(http.CanonicalHeaderKey("X-Forwarded-Proto"), "https")

	imps := []openrtb.Imp{
		{},
		{},
	}
	setImpsImplicitly(httpReq, imps)
	for _, imp := range imps {
		if imp.Secure == nil || *imp.Secure != 1 {
			t.Errorf("imp.Secure should be set to 1 if the request is https. Got %#v", imp.Secure)
		}
	}
}

func TestRefererParsing(t *testing.T) {
	httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	httpReq.Header.Set("Referer", "http://test.mysite.com")
	bidReq := &openrtb.BidRequest{}

	setSiteImplicitly(httpReq, bidReq)

	if bidReq.Site == nil {
		t.Fatalf("bidrequest.site should not be nil.")
	}

	if bidReq.Site.Domain != "mysite.com" {
		t.Errorf("Bad bidrequest.site.domain. Expected mysite.com, got %s", bidReq.Site.Domain)
	}
	if bidReq.Site.Page != "http://test.mysite.com" {
		t.Errorf("Bad bidrequest.site.page. Expected mysite.com, got %s", bidReq.Site.Page)
	}
}

// TestBadStoredRequests tests diagnostic messages for invalid stored requests
func TestBadStoredRequests(t *testing.T) {
	// Need to turn off aliases for bad requests as applying the alias can fail on a bad request before the expected error is reached.
	tests := &getResponseFromDirectory{
		dir:           "sample-requests/invalid-stored",
		payloadGetter: getRequestPayload,
		messageGetter: getMessage,
		expectedCode:  http.StatusBadRequest,
		aliased:       false,
	}
	tests.assert(t)
}

// Test the stored request functionality
func TestStoredRequests(t *testing.T) {
	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	edep := &endpointDeps{&nobidExchange{}, newParamsValidator(t), &mockStoredReqFetcher{}, empty_fetcher.EmptyFetcher{}, empty_fetcher.EmptyFetcher{}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), map[string]string{}, false, []byte{}, openrtb_ext.BidderMap}

	for i, requestData := range testStoredRequests {
		newRequest, errList := edep.processStoredRequests(context.Background(), json.RawMessage(requestData))
		if len(errList) != 0 {
			for _, err := range errList {
				if err != nil {
					t.Errorf("processStoredRequests Error: %s", err.Error())
				} else {
					t.Error("processStoredRequests Error: received nil error")
				}
			}
		}
		expectJson := json.RawMessage(testFinalRequests[i])
		if !jsonpatch.Equal(newRequest, expectJson) {
			t.Errorf("Error in processStoredRequests, test %d failed on compare\nFound:\n%s\nExpected:\n%s", i, string(newRequest), string(expectJson))
		}
	}
}

// TestOversizedRequest makes sure we behave properly when the request size exceeds the configured max.
func TestOversizedRequest(t *testing.T) {
	reqBody := validRequest(t, "site.json")
	deps := &endpointDeps{
		&nobidExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(len(reqBody) - 1)},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList()),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BidderMap,
	}

	req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps.Auction(recorder, req, nil)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Endpoint should return a 400 if the request exceeds the size max.")
	}

	if bytesRead, err := req.Body.Read(make([]byte, 1)); bytesRead != 0 || err != io.EOF {
		t.Errorf("The request body should still be fully read.")
	}
}

// TestRequestSizeEdgeCase makes sure we behave properly when the request size *equals* the configured max.
func TestRequestSizeEdgeCase(t *testing.T) {
	reqBody := validRequest(t, "site.json")
	deps := &endpointDeps{
		&nobidExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(len(reqBody))},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList()),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BidderMap,
	}

	req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps.Auction(recorder, req, nil)

	if recorder.Code != http.StatusOK {
		t.Errorf("Endpoint should return a 200 if the request equals the size max.")
	}

	if bytesRead, err := req.Body.Read(make([]byte, 1)); bytesRead != 0 || err != io.EOF {
		t.Errorf("The request body should have been read to completion.")
	}
}

// TestNoEncoding prevents #231.
func TestNoEncoding(t *testing.T) {
	endpoint, _ := NewEndpoint(
		&mockExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList()),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BidderMap,
	)
	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if !strings.Contains(recorder.Body.String(), "<script></script>") {
		t.Errorf("The Response from the exchange should not be html-encoded")
	}
}

// TestTimeoutParser makes sure we parse tmax properly.
func TestTimeoutParser(t *testing.T) {
	reqJson := json.RawMessage(`{"tmax":22}`)
	timeout := parseTimeout(reqJson, 11*time.Millisecond)
	if timeout != 22*time.Millisecond {
		t.Errorf("Failed to parse tmax properly. Expected %d, got %d", 22*time.Millisecond, timeout)
	}
}

func TestImplicitAMPNoExt(t *testing.T) {
	httpReq, err := http.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	if !assert.NoError(t, err) {
		return
	}

	bidReq := openrtb.BidRequest{
		Site: &openrtb.Site{},
	}
	setSiteImplicitly(httpReq, &bidReq)
	assert.JSONEq(t, `{"amp":0}`, string(bidReq.Site.Ext))
}

func TestImplicitAMPOtherExt(t *testing.T) {
	httpReq, err := http.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	if !assert.NoError(t, err) {
		return
	}

	bidReq := openrtb.BidRequest{
		Site: &openrtb.Site{
			Ext: json.RawMessage(`{"other":true}`),
		},
	}
	setSiteImplicitly(httpReq, &bidReq)
	assert.JSONEq(t, `{"amp":0,"other":true}`, string(bidReq.Site.Ext))
}

func TestExplicitAMP(t *testing.T) {
	httpReq, err := http.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site-amp.json")))
	if !assert.NoError(t, err) {
		return
	}

	bidReq := openrtb.BidRequest{
		Site: &openrtb.Site{
			Ext: json.RawMessage(`{"amp":1}`),
		},
	}
	setSiteImplicitly(httpReq, &bidReq)
	assert.JSONEq(t, `{"amp":1}`, string(bidReq.Site.Ext))
}

// TestContentType prevents #328
func TestContentType(t *testing.T) {
	endpoint, _ := NewEndpoint(
		&mockExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList()),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BidderMap,
	)
	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type should be application/json. Got %s", recorder.Header().Get("Content-Type"))
	}
}

// TestDisabledBidder makes sure we pass when encountering a disabled bidder in the configuration.
func TestDisabledBidder(t *testing.T) {
	reqData, err := ioutil.ReadFile("sample-requests/invalid-whole/unknown-bidder.json")
	if err != nil {
		t.Fatalf("Failed to fetch a valid request: %v", err)
	}
	reqBody := string(getRequestPayload(t, reqData))

	deps := &endpointDeps{
		&nobidExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(len(reqBody))},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList()),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{"unknownbidder": "The biddder 'unknownbidder' has been disabled."},
		false,
		[]byte{},
		openrtb_ext.BidderMap,
	}

	req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps.Auction(recorder, req, nil)

	if recorder.Code != http.StatusOK {
		t.Errorf("Endpoint should return a 200 if the unknown bidder was disabled.")
	}

	if bytesRead, err := req.Body.Read(make([]byte, 1)); bytesRead != 0 || err != io.EOF {
		t.Errorf("The request body should have been read to completion.")
	}
}

func TestValidateImpExtDisabledBidder(t *testing.T) {
	imp := &openrtb.Imp{
		Ext: json.RawMessage(`{"appnexus":{"placement_id":555},"unknownbidder":{"foo":"bar"}}`),
	}
	deps := &endpointDeps{
		&nobidExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(8096)},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList()),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{"unknownbidder": "The biddder 'unknownbidder' has been disabled."},
		false,
		[]byte{},
		openrtb_ext.BidderMap,
	}
	errs := deps.validateImpExt(imp, nil, 0)
	assert.JSONEq(t, `{"appnexus":{"placement_id":555}}`, string(imp.Ext))
	assert.Equal(t, []error{&errortypes.BidderTemporarilyDisabled{Message: "The biddder 'unknownbidder' has been disabled."}}, errs)
}

func validRequest(t *testing.T, filename string) string {
	requestData, err := ioutil.ReadFile("sample-requests/valid-whole/supplementary/" + filename)
	if err != nil {
		t.Fatalf("Failed to fetch a valid request: %v", err)
	}
	return string(requestData)
}

// nobidExchange is a well-behaved exchange which always bids "no bid".
type nobidExchange struct {
	gotRequest *openrtb.BidRequest
}

func (e *nobidExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, ids exchange.IdFetcher, labels pbsmetrics.Labels, categoriesFetcher *stored_requests.CategoryFetcher) (*openrtb.BidResponse, error) {
	e.gotRequest = bidRequest
	return &openrtb.BidResponse{
		ID:    bidRequest.ID,
		BidID: "test bid id",
		NBR:   openrtb.NoBidReasonCodeUnknownError.Ptr(),
	}, nil
}

type brokenExchange struct{}

func (e *brokenExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, ids exchange.IdFetcher, labels pbsmetrics.Labels, categoriesFetcher *stored_requests.CategoryFetcher) (*openrtb.BidResponse, error) {
	return nil, errors.New("Critical, unrecoverable error.")
}

func getMessage(t *testing.T, example []byte) []byte {
	if value, err := jsonparser.GetString(example, "message"); err != nil {
		t.Fatalf("Error parsing root.message from request: %v.", err)
	} else {
		return []byte(value)
	}
	return nil
}

// StoredRequest testing

// Test stored request data

// Stored Requests
// first below is valid JSON
// second below is identical to first but with extra '}' for invalid JSON
var testStoredRequestData = map[string]json.RawMessage{
	"2": json.RawMessage(`{
		"tmax": 500,
		"ext": {
			"prebid": {
				"targeting": {
					"pricegranularity": "low"
				}
			}
		}
	}`),
	"3": json.RawMessage(`{
                "tmax": 500,
                "ext": {
                        "prebid": {
                                "targeting": {
                                        "pricegranularity": "low"
                                }
                        }
                }}
        }`),
}

// Stored Imp Requests
// first below has valid JSON but doesn't match schema
// second below has invalid JSON (missing comma after rubicon accountId entry) but otherwise matches schema
// third below has valid JSON and matches schema
var testStoredImpData = map[string]json.RawMessage{
	"1": json.RawMessage(`{
		"id": "adUnit1",
			"ext": {
				"appnexus": {
					"placementId": "abc",
					"position": "above",
					"reserve": 0.35
				},
				"rubicon": {
					"accountId": "abc"
				}
			}
		}`),
	"7": json.RawMessage(`{
		"id": "adUnit1",
			"ext": {
				"appnexus": {
					"placementId": 12345678,
					"position": "above",
					"reserve": 0.35
				},
				"rubicon": {
					"accountId": 23456789
					"siteId": 113932,
					"zoneId": 535510
				}
			}
		}`),
	"9": json.RawMessage(`{
		"id": "adUnit1",
			"ext": {
				"appnexus": {
					"placementId": 12345678,
					"position": "above",
					"reserve": 0.35
				},
				"rubicon": {
					"accountId": 23456789,
					"siteId": 113932,
					"zoneId": 535510
				}
			}
		}`),
}

// Incoming requests with stored request IDs
var testStoredRequests = []string{
	`{
		"id": "ThisID",
		"imp": [
			{
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						}
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"imp": [
			{
				"id": "adUnit2",
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						}
					},
					"appnexus": {
						"placementId": "def",
						"trafficSourceCode": "mysite.com",
						"reserve": null
					},
					"rubicon": null
				}
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"imp": [
			{
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						}
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"storedrequest": {
					"id": "2"
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"imp": [
			{
				"id": "some-static-imp",
				"video":{
					"mimes":["video/mp4"]
				},
				"ext": {
					"appnexus": {
						"placementId": "abc",
						"position": "below"
					}
				}
			},
			{
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						}
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
				}
			}
		}
	}`,
}

// The expected requests after stored request processing
var testFinalRequests = []string{
	`{
		"id": "ThisID",
		"imp": [
			{
				"id": "adUnit1",
				"ext": {
					"appnexus": {
						"placementId": "abc",
						"position": "above",
						"reserve": 0.35
					},
					"rubicon": {
						"accountId": "abc"
					},
					"prebid": {
						"storedrequest": {
							"id": "1"
						}
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"imp": [
			{
				"id": "adUnit2",
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						}
					},
					"appnexus": {
						"placementId": "def",
						"position": "above",
						"trafficSourceCode": "mysite.com"
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"imp": [
			{
				"id": "adUnit1",
				"ext": {
					"appnexus": {
						"placementId": "abc",
						"position": "above",
						"reserve": 0.35
					},
					"rubicon": {
						"accountId": "abc"
					},
					"prebid": {
						"storedrequest": {
							"id": "1"
						}
					}
				}
			}
		],
		"tmax": 500,
		"ext": {
			"prebid": {
				"targeting": {
					"pricegranularity": "low"
				},
				"storedrequest": {
					"id": "2"
				}
			}
		}
	}`,
	`{
	"id": "ThisID",
	"imp": [
		{
			"id": "some-static-imp",
			"video":{
				"mimes":["video/mp4"]
			},
			"ext": {
				"appnexus": {
					"placementId": "abc",
					"position": "below"
				}
			}
		},
		{
			"id": "adUnit1",
			"ext": {
				"appnexus": {
					"placementId": "abc",
					"position": "above",
					"reserve": 0.35
				},
				"rubicon": {
					"accountId": "abc"
				},
				"prebid": {
					"storedrequest": {
						"id": "1"
					}
				}
			}
		}
	],
	"ext": {
		"prebid": {
			"cache": {
				"markup": 1
			},
			"targeting": {
			}
		}
	}
}`,
}

type mockStoredReqFetcher struct {
}

func (cf mockStoredReqFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	return testStoredRequestData, testStoredImpData, nil
}

type mockExchange struct {
	lastRequest *openrtb.BidRequest
}

func (m *mockExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, ids exchange.IdFetcher, labels pbsmetrics.Labels, categoriesFetcher *stored_requests.CategoryFetcher) (*openrtb.BidResponse, error) {
	m.lastRequest = bidRequest
	return &openrtb.BidResponse{
		SeatBid: []openrtb.SeatBid{{
			Bid: []openrtb.Bid{{
				AdM: "<script></script>",
			}},
		}},
	}, nil
}

func blankAdapterConfig(bidderList []openrtb_ext.BidderName, disabledBidders []string) map[string]config.Adapter {
	adapters := make(map[string]config.Adapter)
	for _, b := range bidderList {
		adapters[string(b)] = config.Adapter{}
	}
	for _, b := range disabledBidders {
		tmp := adapters[b]
		tmp.Disabled = true
		adapters[b] = tmp
	}

	return adapters
}
