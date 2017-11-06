package openrtb_auction

import (
	"testing"
	"github.com/mxmCherry/openrtb"
	"context"
	"net/http/httptest"
	"strings"
	"net/http"
)

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