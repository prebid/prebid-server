package openrtb2

import (
	"testing"
	"github.com/mxmCherry/openrtb"
	"context"
	"net/http/httptest"
	"strings"
	"net/http"
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"
	"bytes"
	"errors"
	"github.com/evanphx/json-patch"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/config"
	"io"
)

const maxSize = 1024 * 256

// TestGoodRequests makes sure that the auction runs properly-formatted bids correctly.
func TestGoodRequests(t *testing.T) {
	endpoint, _ := NewEndpoint(&nobidExchange{}, &bidderParamValidator{}, empty_fetcher.EmptyFetcher(), &config.Configuration{ MaxRequestSize: maxSize })

	for _, requestData := range validRequests {
		request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(requestData))
		recorder := httptest.NewRecorder()
		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status %d. Got %d. Request data was %s", http.StatusOK, recorder.Code, requestData)
			//t.Errorf("Response body was: %s", recorder.Body)
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

// TestBadRequests makes sure we return 400's on bad requests.
func TestBadRequests(t *testing.T) {
	endpoint, _ := NewEndpoint(&nobidExchange{}, &bidderParamValidator{}, empty_fetcher.EmptyFetcher(), &config.Configuration{ MaxRequestSize: maxSize })
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
	_, err := NewEndpoint(nil, &bidderParamValidator{}, empty_fetcher.EmptyFetcher(), &config.Configuration{ MaxRequestSize: maxSize })
	if err == nil {
		t.Errorf("NewEndpoint should return an error when given a nil Exchange.")
	}
}

// TestNilValidator makes sure we fail when given nil for the BidderParamValidator.
func TestNilValidator(t *testing.T) {
	_, err := NewEndpoint(&nobidExchange{}, nil, empty_fetcher.EmptyFetcher(), &config.Configuration{ MaxRequestSize: maxSize })
	if err == nil {
		t.Errorf("NewEndpoint should return an error when given a nil BidderParamValidator.")
	}
}

// TestExchangeError makes sure we return a 500 if the exchange auction fails.
func TestExchangeError(t *testing.T) {
	endpoint, _ := NewEndpoint(&brokenExchange{}, &bidderParamValidator{}, empty_fetcher.EmptyFetcher(), &config.Configuration{ MaxRequestSize: maxSize })
	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequests[0]))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d. Got %d. Input was: %s", http.StatusInternalServerError, recorder.Code, validRequests[0])
	}
}

// Test the stored request functionality
func TestStoredRequests(t *testing.T) {
	edep := &endpointDeps{&nobidExchange{}, &bidderParamValidator{}, &mockStoredReqFetcher{}, &config.Configuration{ MaxRequestSize: maxSize }}

	for i, requestData := range testStoredRequests {
		Request := openrtb.BidRequest{}
		err := json.Unmarshal(json.RawMessage(requestData), &Request)
		if err != nil {
			t.Errorf("Error unmashalling bid request: %s", err.Error())
		}

		errList := edep.processStoredRequests(context.Background(), &Request, json.RawMessage(requestData))
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
		requestJson, err := json.Marshal(Request)
		if err != nil {
			t.Errorf("Error mashalling bid request: %s", err.Error())
		}
		if ! jsonpatch.Equal(requestJson, expectJson) {
			t.Errorf("Error in processStoredRequests, test %d failed on compare\nFound:\n%s\nExpected:\n%s", i, string(requestJson), string(expectJson))
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
		&config.Configuration{ MaxRequestSize: int64(len(reqBody) - 1) },
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
		&config.Configuration{ MaxRequestSize: maxSize })
	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequests[0]))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if !strings.Contains(recorder.Body.String(), "<script></script>") {
		t.Errorf("The Response from the exchange should not be html-encoded")
	}
}

// nobidExchange is a well-behaved exchange which always bids "no bid".
type nobidExchange struct {}

func (e *nobidExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest) (*openrtb.BidResponse, error) {
	return &openrtb.BidResponse{
		ID: bidRequest.ID,
		BidID: "test bid id",
		NBR: openrtb.NoBidReasonCodeUnknownError.Ptr(),
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

type brokenExchange struct {}

func (e *brokenExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest) (*openrtb.BidResponse, error) {
	return nil, errors.New("Critical, unrecoverable error.")
}

func (validator *bidderParamValidator) Schema(name openrtb_ext.BidderName) string {
	return "{}"
}

var validRequests = []string{
	`{
		"id": "some-request-id",
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
					"lengthmax": 20
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
					"lengthmax": 20
				}
			}
		}
	}`,
}

// The expected requests after stored request processing
var testFinalRequests = []string {
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
					"lengthmax": 20
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
					"lengthmax": 20
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

type mockExchange struct {}

func (*mockExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest) (*openrtb.BidResponse, error) {
	return &openrtb.BidResponse{
		SeatBid: []openrtb.SeatBid{{
			Bid: []openrtb.Bid{{
				AdM: "<script></script>",
			}},
		}},
	}, nil
}
