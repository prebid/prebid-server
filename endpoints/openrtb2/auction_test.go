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
)

// TestGoodRequests makes sure that the auction runs properly-formatted bids correctly.
func TestGoodRequests(t *testing.T) {
	endpoint, _ := NewEndpoint(&nobidExchange{}, &bidderParamValidator{})

	for _, requestData := range validRequests {
		request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(requestData))
		recorder := httptest.NewRecorder()
		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status %d. Got %d. Request data was %s", http.StatusOK, recorder.Code, requestData)
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
	endpoint, _ := NewEndpoint(&nobidExchange{}, &bidderParamValidator{})
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
	_, err := NewEndpoint(nil, &bidderParamValidator{})
	if err == nil {
		t.Errorf("NewEndpoint should return an error when given a nil Exchange.")
	}
}

// TestNilValidator makes sure we fail when given nil for the BidderParamValidator.
func TestNilValidator(t *testing.T) {
	_, err := NewEndpoint(&nobidExchange{}, nil)
	if err == nil {
		t.Errorf("NewEndpoint should return an error when given a nil BidderParamValidator.")
	}
}

// TestExchangeError makes sure we return a 500 if the exchange auction fails.
func TestExchangeError(t *testing.T) {
	endpoint, _ := NewEndpoint(&brokenExchange{}, &bidderParamValidator{})
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
	endpoint, _ := NewEndpoint(ex, &bidderParamValidator{})
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

// nobidExchange is a well-behaved exchange which always bids "no bid".
type nobidExchange struct {
	gotRequest *openrtb.BidRequest
}

func (e *nobidExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest) (*openrtb.BidResponse, error) {
	e.gotRequest = bidRequest
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
}
