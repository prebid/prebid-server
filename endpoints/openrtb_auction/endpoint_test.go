package openrtb_auction

import (
	"testing"
	"github.com/mxmCherry/openrtb"
	"context"
	"net/http/httptest"
	"strings"
	"net/http"
	"encoding/json"
)

func TestGoodRequest(t *testing.T) {
	endpoint := &EndpointDeps{
		Exchange: &nobidExchange{},
	}

	reqData := `
{
  "id": "some-request-id",
  "imp": [
    {
      "id": "my-imp-id",
      "banner": {
    	"format": [
    	  {
    	    "w": 300,
    	    "h": 250
    	  },
    	  {
    	    "w": 300,
    	    "h": 600
    	  }
    	]
      },
      "ext": {
        "appnexus": {
          "placementId": "10433394"
        }
      }
    }
  ],
  "test": 1,
  "tmax": 500
}
	`

	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	endpoint.Auction(recorder, request, nil)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status %d. Got %d", http.StatusOK, recorder.Code)
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

func TestBadRequestBody(t *testing.T) {
	endpoint := &EndpointDeps{
		Exchange: &nobidExchange{},
	}

	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader("5"))
	recorder := httptest.NewRecorder()

	endpoint.Auction(recorder, request, nil)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d. Got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestMissingRequestID(t *testing.T) {
	endpoint := &EndpointDeps{
		Exchange: &nobidExchange{},
	}

	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader("{}"))
	recorder := httptest.NewRecorder()

	endpoint.Auction(recorder, request, nil)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d. Got %d", http.StatusBadRequest, recorder.Code)
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