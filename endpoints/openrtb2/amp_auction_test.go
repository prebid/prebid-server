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
		"10": json.RawMessage(validRequest(t, "site.json")),
	}

	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	endpoint, _ := NewAmpEndpoint(&mockAmpExchange{}, &bidderParamValidator{}, &mockAmpStoredReqFetcher{goodRequests}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics)

	for requestID, _ := range goodRequests {
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
	}
}

// TestBadRequests makes sure we return 400's on bad requests.
func TestAmpBadRequests(t *testing.T) {
	badRequests := map[string]json.RawMessage{
		"11": json.RawMessage(validRequest(t, "app.json")),
		"12": json.RawMessage(validRequest(t, "timeout.json")),
	}

	files := fetchFiles(t, "sample-requests/invalid-whole")
	for index, file := range files {
		badRequests[strconv.Itoa(100+index)] = readFile(t, "sample-requests/invalid-whole/"+file.Name())
	}

	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	endpoint, _ := NewEndpoint(&mockAmpExchange{}, &bidderParamValidator{}, &mockAmpStoredReqFetcher{badRequests}, &config.Configuration{MaxRequestSize: maxSize}, theMetrics)
	for requestID, _ := range badRequests {
		request := httptest.NewRequest("GET", fmt.Sprintf("/openrtb2/auction/amp?tag_id=%s", requestID), nil)
		recorder := httptest.NewRecorder()

		endpoint(recorder, request, nil)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d. Got %d. Input was: %s", http.StatusBadRequest, recorder.Code, fmt.Sprintf("/openrtb2/auction/amp?config=%s", requestID))
		}
	}
}

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
