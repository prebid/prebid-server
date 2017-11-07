package openrtb_auction

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

// TestGoodRequest makes sure that the auction runs a properly-formatted bid correctly.
func TestGoodRequest(t *testing.T) {
	endpoint := &endpointDeps{
		ex: &nobidExchange{},
		paramsValidator: &bidderParamValidator{},
	}

	for _, requestData := range validRequests {
		request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(requestData))
		recorder := httptest.NewRecorder()
		endpoint.Auction(recorder, request, nil)

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

func TestBadRequests(t *testing.T) {
	endpoint := &endpointDeps{
		ex: &nobidExchange{},
		paramsValidator: &bidderParamValidator{},
	}
	for _, badRequest := range invalidRequests {
		request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(badRequest))
		recorder := httptest.NewRecorder()

		endpoint.Auction(recorder, request, nil)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d. Got %d. Input was: %s", http.StatusBadRequest, recorder.Code, badRequest)
		}
	}
}

// nobidExchange is a well-behaved exchange so that we can test the endpoint code directly.
type nobidExchange struct {}

func (e *nobidExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest) *openrtb.BidResponse {
	return &openrtb.BidResponse{
		ID: bidRequest.ID,
		BidID: "test bid id",
		NBR: openrtb.NoBidReasonCodeUnknownError.Ptr(),
	}
}

type bidderParamValidator struct{}

func (validator *bidderParamValidator) Validate(name openrtb_ext.BidderName, ext openrtb.RawJSON) error {
  if bytes.Equal(ext, []byte("\"good\"")) {
		return nil
	} else {
		return errors.New("Bidder params failed validation.")
	}
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
	`{"id":"req-id","imp":[]}`,
	`{"id":"req-id","imp":[{}]}`,
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
		"banner":{
			"format":[{"w":30,"h":50}]
		},
		"pmp":{
		  "deals"[{}]
		}
	}]}`,
}