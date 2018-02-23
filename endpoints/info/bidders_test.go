package info

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestBiddersEndpoint(t *testing.T) {
	endpoint := NewBiddersEndpoint()

	req, err := http.NewRequest("GET", "http://prebid-server.com/info/bidders", strings.NewReader(""))
	if err != nil {
		t.Fatalf("Failed to create a GET /info/bidders request: %v", err)
	}

	r := httptest.NewRecorder()
	endpoint(r, req, nil)
	if r.Code != http.StatusOK {
		t.Errorf("GET /info/bidders returned bad status: %d", r.Code)
	}
	if r.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Bad /info/bidders content type. Expected application/json. Got %s", r.Header().Get("Content-Type"))
	}
	bodyBytes := r.Body.Bytes()
	bidderSlice := make([]string, 0, len(openrtb_ext.BidderMap))
	if err := json.Unmarshal(bodyBytes, &bidderSlice); err != nil {
		t.Errorf("Failed to unmarshal /info/bidders response: %v", err)
	}
	for _, bidderName := range bidderSlice {
		if _, ok := openrtb_ext.BidderMap[bidderName]; !ok {
			t.Errorf("Response from /info/bidders contained unexpected BidderName: %s", bidderName)
		}
	}
	if len(bidderSlice) != len(openrtb_ext.BidderMap) {
		t.Errorf("Response from /info/bidders did not match BidderMap. Expected %d elements. Got %d", len(openrtb_ext.BidderMap), len(bidderSlice))
	}
}
