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
	"github.com/prebid/prebid-server/openrtb2_config/empty_fetcher"
)

// TestGoodRequests makes sure that the auction runs properly-formatted bids correctly.
func TestGoodRequests(t *testing.T) {
	endpoint, _ := NewEndpoint(&nobidExchange{}, &bidderParamValidator{}, empty_fetcher.EmptyFetcher(), empty_fetcher.EmptyFetcher())

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
	endpoint, _ := NewEndpoint(&nobidExchange{}, &bidderParamValidator{}, empty_fetcher.EmptyFetcher(), empty_fetcher.EmptyFetcher())
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
	_, err := NewEndpoint(nil, &bidderParamValidator{}, empty_fetcher.EmptyFetcher(), empty_fetcher.EmptyFetcher())
	if err == nil {
		t.Errorf("NewEndpoint should return an error when given a nil Exchange.")
	}
}

// TestNilValidator makes sure we fail when given nil for the BidderParamValidator.
func TestNilValidator(t *testing.T) {
	_, err := NewEndpoint(&nobidExchange{}, nil, empty_fetcher.EmptyFetcher(), empty_fetcher.EmptyFetcher())
	if err == nil {
		t.Errorf("NewEndpoint should return an error when given a nil BidderParamValidator.")
	}
}

// TestExchangeError makes sure we return a 500 if the exchange auction fails.
func TestExchangeError(t *testing.T) {
	endpoint, _ := NewEndpoint(&brokenExchange{}, &bidderParamValidator{}, empty_fetcher.EmptyFetcher(), empty_fetcher.EmptyFetcher())
	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequests[0]))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d. Got %d. Input was: %s", http.StatusInternalServerError, recorder.Code, validRequests[0])
	}
}

// Test the config cache functionality
func TestConfigCache(t *testing.T) {
	edep := &endpointDeps{&nobidExchange{}, &bidderParamValidator{}, &mockConfigFetcher{}, &mockConfigFetcher{}}

	for i, requestData := range testRequestConfigs {
		Request := openrtb.BidRequest{}
		err := json.Unmarshal(json.RawMessage(requestData), &Request)
		if err != nil {
			t.Errorf("Error unmashalling bid request: %s", err.Error())
		}
		errList := edep.processConfigs(context.Background(), &Request)
		if len(errList) != 0 {
			for _, err := range errList {
				if err != nil {
					t.Errorf("processConfigs Error: %s", err.Error())
				} else {
					t.Error("processConfigs Error: recieved nil error")
				}
			}
		}
		expectJson := json.RawMessage(testFinalRequestConfigs[i])
		requestJson, err := json.Marshal(Request)
		if err != nil {
			t.Errorf("Error mashalling bid request: %s", err.Error())
		}
		if ! jsonpatch.Equal(requestJson, expectJson) {
			t.Errorf("Error in processConfigs, test %d failed on compare\nFound:\n%s\nExpected:\n%s", i, string(requestJson), string(expectJson))
		}

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

// Config Cache testing

// Test stored configs
var testConfigs = map[string]json.RawMessage{
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

// Incoming requests with configs
var testRequestConfigs = []string{
	`{
  "id": "ThisID",
  "imp": [
    {
      "ext": {
        "prebid": {
          "managedconfig": "1"
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
          "managedconfig": "1"
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

// The expected requests after config processing
var testFinalRequestConfigs = []string {
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
		  		"managedconfig": "1"
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
          "managedconfig": "1"
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

type mockConfigFetcher struct {
}

func (cf mockConfigFetcher) GetConfigs(ctx context.Context, ids []string) (map[string]json.RawMessage, []error) {
	return testConfigs, nil
}