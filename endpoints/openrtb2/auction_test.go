package openrtb2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evanphx/json-patch"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/rcrowley/go-metrics"
)

const maxSize = 1024 * 256

// TestGoodRequests makes sure that the auction runs properly-formatted bids correctly.
func TestGoodRequests(t *testing.T) {
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList())
	endpoint, _ := NewEndpoint(&nobidExchange{}, &bidderParamValidator{}, empty_fetcher.EmptyFetcher(), &config.Configuration{MaxRequestSize: maxSize}, theMetrics)

	for _, requestData := range validRequests {
		request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(requestData))
		recorder := httptest.NewRecorder()
		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusOK {
			t.Fatalf("Expected status %d. Got %d. Request data was %s\n\nResponse body was: %s", http.StatusOK, recorder.Code, requestData, recorder.Body.String())
		}

		var response openrtb.BidResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Error unmarshalling response: %s", err.Error())
		}

		if response.ID != "some-request-id" {
			t.Errorf("Bad response.id. Expected %s, got %s.", "some-request-id", response.ID)
		}
		if response.BidID != "test bid id" {
			t.Errorf("Bad response.id. Expected %s, got %s.", "test bid id", response.BidID)
		}
		if *response.NBR != openrtb.NoBidReasonCodeUnknownError {
			t.Errorf("Bad response.nbr. Expected %d, got %d.", openrtb.NoBidReasonCodeUnknownError, response.NBR)
		}
	}
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
					"appnexus": "good"
				}
			}
		]
	}`))
	request.AddCookie(&http.Cookie{
		Name:  cookieName,
		Value: mockId,
	})
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList())
	endpoint, _ := NewEndpoint(ex, &bidderParamValidator{}, empty_fetcher.EmptyFetcher(), cfg, theMetrics)
	endpoint(httptest.NewRecorder(), request, nil)

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

	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequests[0]))
	request.AddCookie(&http.Cookie{
		Name:  cookieName,
		Value: mockId,
	})
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList())
	endpoint, _ := NewEndpoint(ex, &bidderParamValidator{}, empty_fetcher.EmptyFetcher(), cfg, theMetrics)
	endpoint(httptest.NewRecorder(), request, nil)

	if ex.lastRequest.User == nil {
		t.Fatalf("The exchange should have received a request with a non-nil user.")
	}

	if ex.lastRequest.User.ID != mockId {
		t.Errorf("Bad User ID. Expected %s, got %s", mockId, ex.lastRequest.User.ID)
	}
}

// TestBadRequests makes sure we return 400's on bad requests.
func TestBadRequests(t *testing.T) {
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList())
	endpoint, _ := NewEndpoint(&nobidExchange{}, &bidderParamValidator{}, empty_fetcher.EmptyFetcher(), &config.Configuration{MaxRequestSize: maxSize}, theMetrics)
	for _, badRequest := range invalidRequests {
		request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(badRequest))
		recorder := httptest.NewRecorder()

		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d. Got %d. Input was: %s", http.StatusBadRequest, recorder.Code, badRequest)
		}
	}
}

// TestNilExchange makes sure we fail when given nil for the Exchange.
func TestNilExchange(t *testing.T) {
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList())
	_, err := NewEndpoint(nil, &bidderParamValidator{}, empty_fetcher.EmptyFetcher(), &config.Configuration{MaxRequestSize: maxSize}, theMetrics)
	if err == nil {
		t.Errorf("NewEndpoint should return an error when given a nil Exchange.")
	}
}

// TestNilValidator makes sure we fail when given nil for the BidderParamValidator.
func TestNilValidator(t *testing.T) {
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList())
	_, err := NewEndpoint(&nobidExchange{}, nil, empty_fetcher.EmptyFetcher(), &config.Configuration{MaxRequestSize: maxSize}, theMetrics)
	if err == nil {
		t.Errorf("NewEndpoint should return an error when given a nil BidderParamValidator.")
	}
}

// TestExchangeError makes sure we return a 500 if the exchange auction fails.
func TestExchangeError(t *testing.T) {
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList())
	endpoint, _ := NewEndpoint(&brokenExchange{}, &bidderParamValidator{}, empty_fetcher.EmptyFetcher(), &config.Configuration{MaxRequestSize: maxSize}, theMetrics)
	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequests[0]))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d. Got %d. Input was: %s", http.StatusInternalServerError, recorder.Code, validRequests[0])
	}
}

// TestUserAgentSetting makes sure we read the User-Agent header if it wasn't defined on the request.
func TestUserAgentSetting(t *testing.T) {
	httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequests[0]))
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
	httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequests[0]))
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

// TestImplicitIPs prevents #230
func TestImplicitIPs(t *testing.T) {
	ex := &nobidExchange{}
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList())
	endpoint, _ := NewEndpoint(ex, &bidderParamValidator{}, &mockStoredReqFetcher{}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics)
	httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequests[0]))
	httpReq.Header.Set("X-Forwarded-For", "123.456.78.90")
	recorder := httptest.NewRecorder()

	endpoint(recorder, httpReq, nil)

	if ex.gotRequest.Device.IP != "123.456.78.90" {
		t.Errorf("Bad device IP. Expected 123.456.78.90, got %s", ex.gotRequest.Device.IP)
	}
}

func TestRefererParsing(t *testing.T) {
	httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequests[0]))
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

// Test valid/invalid DigiTrust functionality
func TestDigiTrust(t *testing.T) {
	for _, requestData := range digiTrustTestRequests {
		bidReq := &openrtb.BidRequest{}
		err := json.Unmarshal(json.RawMessage(requestData), &bidReq)
		if err != nil {
			t.Errorf("Error unmashalling bid request: %s", err.Error())
		}

		err = validateUser(bidReq.User)

		switch bidReq.ID {
		case "request-without-user-obj":
			if err != nil {
				t.Fatalf("validateUser should not return an error due to digitrust.")
			}
		case "request-without-user-ext-obj":
			if err != nil {
				t.Fatalf("validateUser should not return an error due to digitrust.")
			}
		case "request-with-valid-digitrust-obj":
			if err != nil {
				t.Fatalf("validateUser should not return an error due to digitrust.")
			}
		case "request-with-invalid-digitrust-obj":
			if err == nil {
				t.Fatalf("validateUser should return an error due to digitrust.")
			}
		}
	}
}

// Test the stored request functionality
func TestStoredRequests(t *testing.T) {
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList())
	edep := &endpointDeps{&nobidExchange{}, &bidderParamValidator{}, &mockStoredReqFetcher{}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics}

	for i, requestData := range testStoredRequests {
		newRequest, errList := edep.processStoredRequests(context.Background(), json.RawMessage(requestData))
		if len(errList) != 0 {
			for _, err := range errList {
				if err != nil {
					t.Errorf("processStoredRequests Error: %s", err.Error())
				} else {
					t.Error("processStoredRequests Error: recieved nil error")
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
	reqBody := `{"id":"request-id"}`
	deps := &endpointDeps{
		&nobidExchange{},
		&bidderParamValidator{},
		&mockStoredReqFetcher{},
		&config.Configuration{MaxRequestSize: int64(len(reqBody) - 1)},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList()),
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
	reqBody := validRequests[0]
	deps := &endpointDeps{
		&nobidExchange{},
		&bidderParamValidator{},
		&mockStoredReqFetcher{},
		&config.Configuration{MaxRequestSize: int64(len(reqBody))},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList()),
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
		&bidderParamValidator{},
		&mockStoredReqFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList()))
	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequests[0]))
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

// TestContentType prevents #328
func TestContentType(t *testing.T) {
	endpoint, _ := NewEndpoint(
		&mockExchange{},
		&bidderParamValidator{},
		&mockStoredReqFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList()))
	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequests[0]))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type should be application/json. Got %s", recorder.Header().Get("Content-Type"))
	}
}

// nobidExchange is a well-behaved exchange which always bids "no bid".
type nobidExchange struct {
	gotRequest *openrtb.BidRequest
}

func (e *nobidExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, ids exchange.IdFetcher) (*openrtb.BidResponse, error) {
	e.gotRequest = bidRequest
	return &openrtb.BidResponse{
		ID:    bidRequest.ID,
		BidID: "test bid id",
		NBR:   openrtb.NoBidReasonCodeUnknownError.Ptr(),
	}, nil
}

// bidderParamValidator expects the extension format for all bidders to be the JSON string "good".
// Substantive tests for bidder param validation should go in openrtb_ext/bidders_test.go.
type bidderParamValidator struct{}

func (validator *bidderParamValidator) Validate(name openrtb_ext.BidderName, ext openrtb.RawJSON) error {
	if bytes.Equal(ext, []byte("\"good\"")) {
		return nil
	} else {
		return errors.New("Bidder params failed validation.")
	}
}

type brokenExchange struct{}

func (e *brokenExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, ids exchange.IdFetcher) (*openrtb.BidResponse, error) {
	return nil, errors.New("Critical, unrecoverable error.")
}

func (validator *bidderParamValidator) Schema(name openrtb_ext.BidderName) string {
	return "{}"
}

var digiTrustTestRequests = []string{
	`{
		"id": "request-without-user-obj",
		"site": {
			"page": "test.somepage.com"
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
					"appnexus": "good"
				}
			}
		]
	}`,
	`{
		"id": "request-without-user-ext-obj",
		"site": {
			"page": "test.somepage.com"
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
					"appnexus": "good"
				}
			}
		],
		"user": {
			"yob": 1989
		}
	}`,
	`{
		"id": "request-with-valid-digitrust-obj",
		"site": {
			"page": "test.somepage.com"
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
					"appnexus": "good"
				}
			}
		],
		"user": {
			"yob": 1989,
			"ext": {
				"digitrust": {
					"id": "sample-digitrust-id",
					"keyv": 1,
					"pref": 0
				}
			}
		}
	}`,
	`{
		"id": "request-with-invalid-digitrust-obj",
		"site": {
			"page": "test.somepage.com"
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
					"appnexus": "good"
				}
			}
		],
		"user": {
			"yob": 1989,
			"ext": {
				"digitrust": {
					"id": "sample-digitrust-id",
					"keyv": 1,
					"pref": 1
				}
			}
		}
	}`,
}

var validRequests = []string{
	`{
		"id": "some-request-id",
		"site": {
			"page": "test.somepage.com"
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
					"appnexus": "good"
				}
			}
		],
		"ext": {
			"prebid": {
				"targeting": {
					"pricegranularity": "low"
				},
				"cache": {
					"bids": {}
				}
			}
		}
	}`,
	`{
		"id": "some-request-id",
		"app": { },
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
					"appnexus": "good"
				}
			}
		]
	}`,
	`{
		"id": "some-request-id",
		"app": { },
		"tmax": 500,
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
					"appnexus": "good"
				}
			}
		]
	}`,
	`{
		"id": "some-request-id",
		"site": {
			"page": "test.somepage.com"
		},
		"imp": [
			{
				"id": "my-imp-id",
				"video": {
					"mimes":["video/mp4"]
				},
				"ext": {
					"unknown": "good"
				}
			}
		],
		"ext": {
			"prebid": {
				"aliases": {
					"unknown": "appnexus"
				}
			}
		}
	}`,
}

var invalidRequests = []string{
	"5",
	"6.3",
	"null",
	"false",
	"",
	"[]",
	"{}",
	`{"id":"req-id"}`,
	`{"id":"req-id","tmax":-2}`,
	`{"id":"req-id","imp":[]}`,
	`{"id":"req-id","imp":[{}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"metric": [{}]
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id"
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"banner":null
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"banner":{
			"wmin":50
		}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"banner":{
			"wmax":50
		}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"banner":{
			"hmin":50
		}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"banner":{
			"hmax":50
		}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"banner":{
			"format":[]
		}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"banner":{
			"format":[{}]
		}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"banner":{
			"format":[{"w":30,"wratio":23}]
		}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"banner":{
			"format":[{"w":30,"h":0}]
		}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"banner":{
			"format":[{"wratio":30}]
		}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"video":{}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"video":{
			"mimes":[]
		}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"audio":{
			"mimes":[]
		}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"native":{}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"video":{
			"mimes":["video/mp4"]
		},
		"pmp":{
			"deals":[{"private_auction":1, "id":""}]
		}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"video":{
			"mimes":["video/mp4"]
		},
		"ext": {}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"audio":{
			"mimes":["video/mp4"]
		},
		"ext": {
			"noBidderShouldEverHaveThisName": {
				"bogusParam":5
			}
		}
	}]}`,
	`{"id":"req-id","imp":[{
		"id":"imp-id",
		"audio":{
			"mimes":["video/mp4"]
		},
		"ext": {
			"appnexus": "invalidParams"
		}
	}]}`,
	`{"id":"req-id",
		"imp":[{
			"id":"imp-id",
			"video":{
				"mimes":["video/mp4"]
			},
			"ext": {
				"appnexus": "good"
			}
		}]}`,
	`{"id":"req-id",
		"site": {},
		"imp":[{
			"id":"imp-id",
			"video":{
				"mimes":["video/mp4"]
			},
			"ext": {
				"appnexus": "good"
			}
		}]
	}`,
	`{"id":"req-id",
		"site": {"page":"test.mysite.com"},
		"app": {},
		"imp":[{
			"id":"imp-id",
			"video":{
				"mimes":["video/mp4"]
			},
			"ext": {
				"appnexus": "good"
			}
		}]
	}`,
	`{"id": "some-request-id",
		"site": {"page": "test.somepage.com"},
		"imp": [{
			"id":"imp-id",
			"video":{
				"mimes":["video/mp4"]
			},
			"ext": {
				"appnexus": "good"
			}
		}],
		"ext": {
			"prebid": {
				"storedrequest": {
					"id": 13
				}
			}
		}
	}`,
	`{
		"id": "some-request-id",
		"site": {"page": "test.somepage.com"},
		"imp": [{
			"id": "my-imp-id",
			"video": {
				"mimes":["video/mp4"]
			},
			"ext": {
				"appnexus": "good"
			}
		}],
		"ext": {
			"prebid": {
				"cache": {}
			}
		}
	}`,
	`{
		"id": "some-request-id",
		"site": {
			"page": "test.somepage.com"
		},
		"imp": [
			{
				"id": "my-imp-id",
				"video": {
					"mimes":["video/mp4"]
				},
				"ext": {
					"unknown": "good"
				}
			}
		],
		"ext": {
			"prebid": {
				"aliases": {
					"unknown": "other-unknown"
				}
			}
		}
	}`,
	`{
		"id": "some-request-id",
		"site": {
			"page": "test.somepage.com"
		},
		"imp": [
			{
				"id": "my-imp-id",
				"video": {
					"mimes":["video/mp4"]
				},
				"ext": {
					"appnexus": "good"
				}
			}
		],
		"ext": {
			"prebid": {
				"aliases": {
					"appnexus": "appnexus"
				}
			}
		}
	}`,
}

// StoredRequest testing

// Test stored request data
var testStoredRequestData = map[string]json.RawMessage{
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
	"": json.RawMessage(""),
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

func (cf mockStoredReqFetcher) FetchRequests(ctx context.Context, ids []string) (map[string]json.RawMessage, []error) {
	return testStoredRequestData, nil
}

type mockExchange struct {
	lastRequest *openrtb.BidRequest
}

func (m *mockExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, ids exchange.IdFetcher) (*openrtb.BidResponse, error) {
	m.lastRequest = bidRequest
	return &openrtb.BidResponse{
		SeatBid: []openrtb.SeatBid{{
			Bid: []openrtb.Bid{{
				AdM: "<script></script>",
			}},
		}},
	}, nil
}
