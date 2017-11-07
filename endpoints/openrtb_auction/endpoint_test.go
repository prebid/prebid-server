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

// TestGoodRequest makes sure that the auction runs a properly-formatted bid correctly.
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
          "placementId": "10433394"
        }
      }
    }
  ],
  "test": 1,
  "tmax": 500
}`

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

// TestBadRequestBody makes sure we return 400s if we cant turn the request body into an openrtb.BidRequest
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

// TestInvalidRequestBody makes sure we return 400s if the body has a valid form, but invalid contents.
func TestInvalidRequestBody(t *testing.T) {
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

// TestMissingRequestID makes sure we return 400s if the request has no ID.
// This also intends to prove that we return 400s if the request doesn't pass validation in our code.
// This will make future tests easy
func TestMissingRequestID(t *testing.T) {
	endpoint := &EndpointDeps{
		Exchange: &nobidExchange{},
	}

	reqData := `
{
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
      "ext": {
        "appnexus": {
          "placementId": "10433394"
        }
      }
    }
  ],
  "test": 1,
  "tmax": 500`

	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(reqData))
	recorder := httptest.NewRecorder()

	endpoint.Auction(recorder, request, nil)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d. Got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestNoImpId(t *testing.T) {
	req := openrtb.BidRequest{}
	if err := validateRequest(&req); err == nil {
		t.Error("We should require bidrequest.imp[i].id to be defined.")
	}
}

func TestNoImps(t *testing.T) {
	req := openrtb.BidRequest{
		ID: "some-id",
	}
	if err := validateRequest(&req); err == nil {
		t.Error("We should require a BidRequest to have at least one Imp.")
	}
}

func TestInvalidImp(t *testing.T) {
	req := openrtb.BidRequest{
		ID: "some-id",
		Imp: []openrtb.Imp{{}},
	}
	if err := validateRequest(&req); err == nil {
		t.Error("The validateRequest function should reject inputs with malformed imp values.")
	}
}

func TestImpMetric(t *testing.T) {
	imp := openrtb.Imp{
		ID: "imp-id",
		Metric: []openrtb.Metric{{}},
	}

	if err := validateImp(&imp, 0); err == nil {
		t.Error("We should reject requests which define imp.metric[j].")
	}
}

func TestSomeMediaTypes(t *testing.T) {
	imp := openrtb.Imp{
		ID: "imp-id",
	}
	if err := validateImp(&imp, 0); err == nil {
		t.Error("We should require imps to define one of banner, video, audio, or native.")
	}
}

func TestInvalidBanner(t *testing.T) {
	imp := openrtb.Imp{
		ID: "imp-id",
		Banner: &openrtb.Banner{
			WMin: 15,
		},
	}
	if err := validateImp(&imp, 0); err == nil {
		t.Error("We should reject imps with an invalid banner object.")
	}
}

func TestBannerWmax(t *testing.T) {
	banner := openrtb.Banner{
		WMax: 15,
	}
	if err := validateBanner(&banner, 0); err == nil {
		t.Error("We should reject banners which define wmax.")
	}
}

func TestBannerHMin(t *testing.T) {
	banner := openrtb.Banner{
		HMin: 15,
	}
	if err := validateBanner(&banner, 0); err == nil {
		t.Error("We should reject banners which define him.")
	}
}

func TestBannerHMax(t *testing.T) {
	banner := openrtb.Banner{
		HMax: 15,
	}
	if err := validateBanner(&banner, 0); err == nil {
		t.Error("We should reject banners which define hmax.")
	}
}

func TestBannerFormat(t *testing.T) {
	banner := openrtb.Banner{
		Format: []openrtb.Format{{
			W: 1,
			H: 2,
			WMin: 14,
			WRatio: 15,
			HRatio: 20,
		}},
	}
	if err := validateBanner(&banner, 0); err == nil {
		t.Error("We should reject banners with invalid formats.")
	}
}

func TestIncompleteFixedFormat(t *testing.T) {
	format := openrtb.Format{
		W: 1,
	}
	if err := validateFormat(&format, 0, 0); err == nil {
		t.Error("We should reject fixed formats which exclude width or height.")
	}
}

func TestIncompleteFlexFormat(t *testing.T) {
	format := openrtb.Format{
		WMin: 2,
	}
	if err := validateFormat(&format, 0, 0); err == nil {
		t.Error("We should reject flex formats which exclude the required properties.")
	}
}

func TestEmptyFormat(t *testing.T) {
	format := openrtb.Format{}
	if err := validateFormat(&format, 0, 0); err == nil {
		t.Error("We should reject empty formats.")
	}
}

func TestInvalidVideo(t *testing.T) {
	imp := openrtb.Imp{
		ID: "imp-id",
		Video: &openrtb.Video{},
	}
	if err := validateImp(&imp, 0); err == nil {
		t.Error("We should reject imps which define no acceptable video MIME types.")
	}
}

func TestInvalidAudio(t *testing.T) {
	imp := openrtb.Imp{
		ID: "imp-id",
		Audio: &openrtb.Audio{},
	}
	if err := validateImp(&imp, 0); err == nil {
		t.Error("We should reject imps which define no acceptable audio MIME types.")
	}
}

func TestEmptyNative(t *testing.T) {
	imp := openrtb.Imp{
		ID: "imp-id",
		Native: &openrtb.Native{},
	}
	if err := validateImp(&imp, 0); err == nil {
		t.Error("We should reject imps which define empty native request strings.")
	}
}

func TestInvalidPmps(t *testing.T) {
	imp := openrtb.Imp{
		ID: "imp-id",
		Video: &openrtb.Video{
			MIMEs: []string{"abc"},
		},
		PMP: &openrtb.PMP{
			Deals: []openrtb.Deal{{}},
		},
	}
	if err := validateImp(&imp, 0); err == nil {
		t.Error("We should reject imps which define malformed PMPs.")
	}
}

func TestNilPmps(t *testing.T) {
	if err := validatePmp(nil, 0); err != nil {
		t.Error("We should allow requests which don't define a PMP.")
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