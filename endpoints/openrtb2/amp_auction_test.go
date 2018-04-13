package openrtb2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/rcrowley/go-metrics"
)

// From auction_test.go
// const maxSize = 1024 * 256

// TestGoodRequests makes sure that the auction runs properly-formatted stored bids correctly.
func TestGoodAmpRequests(t *testing.T) {
	goodRequests := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "aliased-buyeruids.json")),
		"2": json.RawMessage(validRequest(t, "aliases.json")),
		"3": json.RawMessage(validRequest(t, "app.json")),
		"4": json.RawMessage(validRequest(t, "digitrust.json")),
		"5": json.RawMessage(validRequest(t, "gdpr-no-consentstring.json")),
		"6": json.RawMessage(validRequest(t, "gdpr.json")),
		"7": json.RawMessage(validRequest(t, "site.json")),
		"8": json.RawMessage(validRequest(t, "timeout.json")),
		"9": json.RawMessage(validRequest(t, "user.json")),
	}

	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	endpoint, _ := NewAmpEndpoint(&mockAmpExchange{}, &bidderParamValidator{}, &mockAmpStoredReqFetcher{goodRequests}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics, analytics.NewPBSAnalytics(&config.Analytics{}))

	for requestID := range goodRequests {
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=%s", requestID), nil)
		recorder := httptest.NewRecorder()
		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status %d. Got %d. Request config ID was %s", http.StatusOK, recorder.Code, requestID)
			t.Errorf("Response body was: %s", recorder.Body)
			t.Errorf("Request was: %s", string(goodRequests[requestID]))
		}

		var response AmpResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Error unmarshalling response: %s", err.Error())
		}

		if response.Targeting == nil || len(response.Targeting) == 0 {
			t.Errorf("Bad response, no targeting data.\n Response was: %v", recorder.Body)
		}
		if len(response.Targeting) != 3 {
			t.Errorf("Bad targeting data. Expected 3 keys, got %d.", len(response.Targeting))
		}

		if response.Debug != nil {
			t.Errorf("Debug present but not requested")
		}
	}
}

// TestBadRequests makes sure we return 400's on bad requests.
func TestAmpBadRequests(t *testing.T) {
	files := fetchFiles(t, "sample-requests/invalid-whole")
	badRequests := make(map[string]json.RawMessage, len(files))
	for index, file := range files {
		badRequests[strconv.Itoa(100+index)] = readFile(t, "sample-requests/invalid-whole/"+file.Name())
	}

	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	endpoint, _ := NewEndpoint(&mockAmpExchange{}, &bidderParamValidator{}, &mockAmpStoredReqFetcher{badRequests}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics, analytics.NewPBSAnalytics(&config.Analytics{}))
	for requestID := range badRequests {
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=%s", requestID), nil)
		recorder := httptest.NewRecorder()

		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d. Got %d. Input was: %s", http.StatusBadRequest, recorder.Code, fmt.Sprintf("/openrtb2/auction/amp?config=%s", requestID))
		}
	}
}

// TestAmpDebug makes sure we get debug information back when requested
func TestAmpDebug(t *testing.T) {
	requests := map[string]json.RawMessage{
		"1": json.RawMessage(validRequest(t, "app.json")),
		"2": json.RawMessage(validRequest(t, "site.json")),
	}

	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	endpoint, _ := NewAmpEndpoint(&mockAmpExchange{}, &bidderParamValidator{}, &mockAmpStoredReqFetcher{requests}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics, analytics.NewPBSAnalytics(&config.Analytics{}))

	for requestID := range requests {
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=%s&debug=1", requestID), nil)
		recorder := httptest.NewRecorder()
		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status %d. Got %d. Request config ID was %s", http.StatusOK, recorder.Code, requestID)
			t.Errorf("Response body was: %s", recorder.Body)
			t.Errorf("Request was: %s", string(requests[requestID]))
		}

		var response AmpResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Error unmarshalling response: %s", err.Error())
		}

		if response.Targeting == nil || len(response.Targeting) == 0 {
			t.Errorf("Bad response, no targeting data.\n Response was: %v", recorder.Body)
		}
		if len(response.Targeting) != 3 {
			t.Errorf("Bad targeting data. Expected 3 keys, got %d.", len(response.Targeting))
		}

		if response.Debug == nil {
			t.Errorf("Debug requested but not present")
		}
	}
}

// Prevents #452
func TestAmpTargetingDefaults(t *testing.T) {
	req := &openrtb.BidRequest{}
	if errs := defaultRequestExt(req); len(errs) != 0 {
		t.Fatalf("Unexpected error defaulting request.ext for AMP: %v", errs)
	}

	var extRequest openrtb_ext.ExtRequest
	if err := json.Unmarshal(req.Ext, &extRequest); err != nil {
		t.Fatalf("Unexpected error unmarshalling defaulted request.ext for AMP: %v", err)
	}
	if extRequest.Prebid.Targeting == nil {
		t.Fatal("AMP defaults should set request.ext.targeting")
	}
	if !extRequest.Prebid.Targeting.IncludeWinners {
		t.Error("AMP defaults should set request.ext.targeting.includewinners to true")
	}
	if extRequest.Prebid.Targeting.PriceGranularity != openrtb_ext.PriceGranularityMedium {
		t.Error("AMP defaults should set request.ext.targeting.pricegranularity to medium")
	}
}

type mockAmpStoredReqFetcher struct {
	data map[string]json.RawMessage
}

func (cf *mockAmpStoredReqFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	return cf.data, nil, nil
}

type mockAmpExchange struct {
	lastRequest *openrtb.BidRequest
}

func (m *mockAmpExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, ids exchange.IdFetcher, labels pbsmetrics.Labels) (*openrtb.BidResponse, error) {
	m.lastRequest = bidRequest

	response := &openrtb.BidResponse{
		SeatBid: []openrtb.SeatBid{{
			Bid: []openrtb.Bid{{
				AdM: "<script></script>",
				Ext: openrtb.RawJSON(`{ "prebid": {"targeting": { "hb_pb": "1.20", "hb_appnexus_pb": "1.20", "hb_cache_id": "some_id"}}}`),
			}},
		}},
	}

	if bidRequest.Test == 1 {
		response.Ext = openrtb.RawJSON(`{"debug": {"httpcalls": {}, "resolvedrequest": {}}}`)
	}

	return response, nil
}
