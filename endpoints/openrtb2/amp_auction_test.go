package openrtb2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/rcrowley/go-metrics"
)

// From auction_test.go
// const maxSize = 1024 * 256

// TestGoodRequests makes sure that the auction runs properly-formatted stored bids correctly.
func TestGoodAmpRequests(t *testing.T) {
	testAmpStoredRequestData := map[string]json.RawMessage{
		"10":  json.RawMessage(validRequest(t, "site.json")),
		"11":  json.RawMessage(validRequest(t, "app.json")),
		"12":  json.RawMessage(validRequest(t, "timeout.json")),
		"100": json.RawMessage("5"),
		"101": json.RawMessage("5"),
		"102": json.RawMessage("5"),
		"103": json.RawMessage("5"),
		"104": json.RawMessage("5"),
		"105": json.RawMessage("5"),
		"106": json.RawMessage("5"),
		"107": json.RawMessage("5"),
		"108": json.RawMessage("5"),
		"109": json.RawMessage("5"),
		"110": json.RawMessage("5"),
		"111": json.RawMessage("5"),
		"112": json.RawMessage("5"),
		"113": json.RawMessage("5"),
		"114": json.RawMessage("5"),
		"115": json.RawMessage("5"),
		"116": json.RawMessage("5"),
		"117": json.RawMessage("5"),
		"118": json.RawMessage("5"),
		"119": json.RawMessage("5"),
		"120": json.RawMessage("5"),
		"121": json.RawMessage("5"),
		"122": json.RawMessage("5"),
		"123": json.RawMessage("5"),
		"124": json.RawMessage("5"),
		"125": json.RawMessage("5"),
		"126": json.RawMessage("5"),
		"127": json.RawMessage("5"),
		"128": json.RawMessage("5"),
		"129": json.RawMessage("5"),
		"130": json.RawMessage("5"),
		"131": json.RawMessage("5"),
		"132": json.RawMessage("5"),
		"133": json.RawMessage("5"),
		"134": json.RawMessage("5"),
	}

	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), exchange.AdapterList())
	endpoint, _ := NewAmpEndpoint(&mockAmpExchange{}, &bidderParamValidator{}, &mockAmpStoredReqFetcher{testAmpStoredRequestData}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics)

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

type mockAmpStoredReqFetcher struct {
	data map[string]json.RawMessage
}

func (cf *mockAmpStoredReqFetcher) FetchRequests(ctx context.Context, ids []string) (map[string]json.RawMessage, []error) {
	return cf.data, nil
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
