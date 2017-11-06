package exchange

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"github.com/mxmCherry/openrtb"
	"context"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestNewExchange(t *testing.T) {
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	// Just match the counts
	e := NewExchange(server.Client()).(*exchange)
	if len(e.adapters) != len(e.adapterMap) {
		t.Errorf("Exchange initialized, but adapter list doesn't match adapter map (%d - %d)", len(e.adapters), len(e.adapterMap))
	}
	// Test that all adapters are in the map and not repeated
	tmp := make(map[openrtb_ext.BidderName]int)
	for _, a := range e.adapters {
		_, ok := tmp[a]
		if ok {
			t.Errorf("Exchange.adapters repeats value %s", a)
		}
		tmp[a] = 1
		_, ok = e.adapterMap[a]
		if !ok {
			t.Errorf("Exchange.adapterMap missing adpater %s", a)
		}
	}
}

func TestGetAllBids(t *testing.T) {

}

type mockAdapter struct {

}

func (a *mockAdapter) Bid(ctx context.Context, request *openrtb.BidRequest) (*adapters.PBSOrtbSeatBid, []error) {
	seatBid := adapters.PBSOrtbSeatBid{}
	errs := make([]error,0,1)

	return &seatBid, errs
}

func mockHandler(statusCode int, getBody string, postBody string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		if r.Method == "GET" {
			w.Write([]byte(getBody))
		} else {
			w.Write([]byte(postBody))
		}
	})
}
