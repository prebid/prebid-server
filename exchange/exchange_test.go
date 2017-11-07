package exchange

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"github.com/mxmCherry/openrtb"
	"context"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"time"
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
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	e := NewDummyExchange(server.Client()).(*exchange)

	sb1 := new(adapters.PBSOrtbSeatBid)
	sb1.Bids = make([]*adapters.PBSOrtbBid, 2)
	sb1.Bids[0] = new(adapters.PBSOrtbBid)
	sb1.Bids[1] = new(adapters.PBSOrtbBid)
	sb1.Bids[0].Bid = new(openrtb.Bid)
	sb1.Bids[1].Bid = new(openrtb.Bid)

	a1 := e.adapterMap[BidderDummy].(*mockAdapter)
	a1.seatBid = sb1
	a1.delay = 10 * time.Millisecond
}

type mockAdapter struct {
	seatBid *adapters.PBSOrtbSeatBid
	errs []error
	delay time.Duration
}

func (a *mockAdapter) Bid(ctx context.Context, request *openrtb.BidRequest) (*adapters.PBSOrtbSeatBid, []error) {
	time.Sleep(a.delay)
	return a.seatBid, a.errs
}

const (
	BidderDummy openrtb_ext.BidderName = "dummy"
	BidderDummy2 openrtb_ext.BidderName = "dummy2"
	BidderDummy3 openrtb_ext.BidderName = "dummy3"
)
// Tester is responsible for filling bid results into the adapters
func NewDummyExchange(client *http.Client) Exchange {
	e := new(exchange)
	a := new(mockAdapter)
	a.errs = make([]error, 0, 5)

	b := new(mockAdapter)
	b.errs = make([]error, 0, 5)

	c := new(mockAdapter)
	c.errs = make([]error, 0, 5)

	e.adapterMap = map[openrtb_ext.BidderName]adapters.Bidder{
		BidderDummy: a,
		BidderDummy2: b,
		BidderDummy3: c,
	}

	e.adapters = make([]openrtb_ext.BidderName, 0, len(e.adapterMap))
	for a, _ := range e.adapterMap {
		e.adapters = append(e.adapters, a)
	}
	return e
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
