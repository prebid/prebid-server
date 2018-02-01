package openrtb2

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/rcrowley/go-metrics"
	"net/http"
	"net/http/httptest"
	"testing"
)

// From auction_test.go
// const maxSize = 1024 * 256

// TestGoodRequests makes sure that the auction runs properly-formatted stored bids correctly.
func TestGoodAmpRequests(t *testing.T) {
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList())
	endpoint, _ := NewAmpEndpoint(&mockAmpExchange{}, &bidderParamValidator{}, &mockAmpStoredReqFetcher{}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics)

	for _, requestID := range storedValidRequests {
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=%s", requestID), nil)
		recorder := httptest.NewRecorder()
		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status %d. Got %d. Request config ID was %s", http.StatusOK, recorder.Code, requestID)
			t.Errorf("Response body was: %s", recorder.Body)
			t.Errorf("Request was: %s", testAmpStoredRequestData[requestID])
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
	}
}

// TestBadRequests makes sure we return 400's on bad requests.
func TestAmpBadRequests(t *testing.T) {
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList())
	endpoint, _ := NewEndpoint(&mockAmpExchange{}, &bidderParamValidator{}, empty_fetcher.EmptyFetcher(), &config.Configuration{MaxRequestSize: maxSize}, theMetrics)
	for _, requestID := range storedInvalidRequests {
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=%s", requestID), nil)
		recorder := httptest.NewRecorder()

		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d. Got %d. Input was: %s", http.StatusBadRequest, recorder.Code, fmt.Sprintf("/openrtb2/auction/amp?config=%s", requestID))
		}
	}
}

// StoredRequest testing

var storedValidRequests = []string{"10"}
var storedInvalidRequests = []string{"11", "12", "100", "101", "102", "103", "104", "105", "106", "107", "108", "109",
	"110", "111", "112", "113", "114", "115", "116", "117", "118", "119",
	"120", "121", "122", "123", "124", "125", "126", "127", "128", "129",
	"103", "131", "132", "133", "134"}

// Test stored request data
var testAmpStoredRequestData = map[string]json.RawMessage{
	"10":  json.RawMessage(validRequests[0]),
	"11":  json.RawMessage(validRequests[1]),
	"12":  json.RawMessage(validRequests[2]),
	"100": json.RawMessage(invalidRequests[0]),
	"101": json.RawMessage(invalidRequests[0]),
	"102": json.RawMessage(invalidRequests[0]),
	"103": json.RawMessage(invalidRequests[0]),
	"104": json.RawMessage(invalidRequests[0]),
	"105": json.RawMessage(invalidRequests[0]),
	"106": json.RawMessage(invalidRequests[0]),
	"107": json.RawMessage(invalidRequests[0]),
	"108": json.RawMessage(invalidRequests[0]),
	"109": json.RawMessage(invalidRequests[0]),
	"110": json.RawMessage(invalidRequests[0]),
	"111": json.RawMessage(invalidRequests[0]),
	"112": json.RawMessage(invalidRequests[0]),
	"113": json.RawMessage(invalidRequests[0]),
	"114": json.RawMessage(invalidRequests[0]),
	"115": json.RawMessage(invalidRequests[0]),
	"116": json.RawMessage(invalidRequests[0]),
	"117": json.RawMessage(invalidRequests[0]),
	"118": json.RawMessage(invalidRequests[0]),
	"119": json.RawMessage(invalidRequests[0]),
	"120": json.RawMessage(invalidRequests[0]),
	"121": json.RawMessage(invalidRequests[0]),
	"122": json.RawMessage(invalidRequests[0]),
	"123": json.RawMessage(invalidRequests[0]),
	"124": json.RawMessage(invalidRequests[0]),
	"125": json.RawMessage(invalidRequests[0]),
	"126": json.RawMessage(invalidRequests[0]),
	"127": json.RawMessage(invalidRequests[0]),
	"128": json.RawMessage(invalidRequests[0]),
	"129": json.RawMessage(invalidRequests[0]),
	"130": json.RawMessage(invalidRequests[0]),
	"131": json.RawMessage(invalidRequests[0]),
	"132": json.RawMessage(invalidRequests[0]),
	"133": json.RawMessage(invalidRequests[0]),
	"134": json.RawMessage(invalidRequests[0]),
}

type mockAmpStoredReqFetcher struct {
}

func (cf mockAmpStoredReqFetcher) FetchRequests(ctx context.Context, ids []string) (map[string]json.RawMessage, []error) {
	return testAmpStoredRequestData, nil
}

type mockAmpExchange struct {
	lastRequest *openrtb.BidRequest
}

func (m *mockAmpExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, ids exchange.IdFetcher) (*openrtb.BidResponse, error) {
	m.lastRequest = bidRequest
	return &openrtb.BidResponse{
		SeatBid: []openrtb.SeatBid{{
			Bid: []openrtb.Bid{{
				AdM: "<script></script>",
				Ext: openrtb.RawJSON(`{ "prebid": {"targeting": { "hb_pb": "1.20", "hb_appnexus_pb": "1.20", "hb_cache_id": "some_id"}}}`),
			}},
		}},
	}, nil
}
