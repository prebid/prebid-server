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
	"encoding/json"
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

func TestHoldAuction(t *testing.T) {
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	e := NewDummyExchange(server.Client()).(*exchange)
	mockAdapterConfig1(e.adapterMap[BidderDummy].(*mockAdapter))
	mockAdapterConfig2(e.adapterMap[BidderDummy2].(*mockAdapter))
	mockAdapterConfig3(e.adapterMap[BidderDummy3].(*mockAdapter))

	// Very simple Bid request. The dummy bidders know what to do.
	bidRequest := new(openrtb.BidRequest)
	bidRequest.ID = "This Bid"
	bidRequest.Imp = make([]openrtb.Imp, 2)

	// Need extensions for all the bidders so we know to hold auctions for them.
	impExt := make(map[string]map[string]string)
	impExt["dummy"] = make(map[string]string)
	impExt["dummy2"] = make(map[string]string)
	impExt["dummy3"] = make(map[string]string)
	b, _ := json.Marshal(impExt)
	bidRequest.Imp[0].Ext = b
	bidRequest.Imp[1].Ext = b

	bidResponse := e.HoldAuction(ctx, bidRequest)
	bidResponseExt := new(openrtb_ext.ExtBidResponse)
	_ = json.Unmarshal(bidResponse.Ext, bidResponseExt)

	if len(bidResponseExt.ResponseTimeMillis) != 3 {
		t.Errorf("HoldAuction: Did not find 3 response times. Found %d instead", len(bidResponseExt.ResponseTimeMillis))
	}
	if len(bidResponse.SeatBid) != 3 {
		t.Errorf("HoldAuction: Expected 3 SeatBids, found %d instead", len(bidResponse.SeatBid))
	}
	if bidResponse.NBR != nil {
		t.Errorf("HoldAuction: Found invalid auction flag in response: %d", *bidResponse.NBR)
	}
	// Find the indexes of the bidders, as they should be scrambled
	// Set initial value to -1 so we error out if bidders are not found.
	var (
		dummy1 = -1
		dummy3 = -1
		)
	for i, sb := range bidResponse.SeatBid {
		if sb.Seat == "dummy" { dummy1 = i }
		if sb.Seat == "dummy3" { dummy3 = i }
	}
	if len(bidResponse.SeatBid[dummy1].Bid) != 2 {
		t.Errorf("HoldAuction: Expected 2 bids from dummy bidder, found %d instead", len(bidResponse.SeatBid[dummy1].Bid))
	}
	if len(bidResponse.SeatBid[dummy3].Bid) != 1 {
		t.Errorf("HoldAuction: Expected 2 bids from dummy bidder, found %d instead", len(bidResponse.SeatBid[dummy1].Bid))
	}

}

func TestGetAllBids(t *testing.T) {
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	e := NewDummyExchange(server.Client()).(*exchange)
	mockAdapterConfig1(e.adapterMap[BidderDummy].(*mockAdapter))
	mockAdapterConfig2(e.adapterMap[BidderDummy2].(*mockAdapter))
	mockAdapterConfig3(e.adapterMap[BidderDummy3].(*mockAdapter))

	cleanRequests := make(map[openrtb_ext.BidderName]*openrtb.BidRequest)
	adapterExtra := make(map[openrtb_ext.BidderName]*seatResponseExtra)
	adapterBids := e.GetAllBids(ctx, e.adapters, cleanRequests, adapterExtra)

	if len(adapterBids[BidderDummy].Bids) != 2 {
		t.Errorf("GetAllBids failed to get 2 bids from BidderDummy, found %d instead", len(adapterBids[BidderDummy].Bids))
	}
	if adapterBids[BidderDummy].Bids[0].Bid.ID != "1234567890" {
		t.Errorf("GetAllBids failed to get the first bid of BidderDummy")
	}
	if adapterBids[BidderDummy3].Bids[0].Bid.ID != "MyBid" {
		t.Errorf("GetAllBids failed to get the bid from BidderDummy3")
	}
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

func mockAdapterConfig1(a *mockAdapter) {

	sb1 := new(adapters.PBSOrtbSeatBid)
	sb1.Bids = make([]*adapters.PBSOrtbBid, 2)
	sb1.Bids[0] = new(adapters.PBSOrtbBid)
	sb1.Bids[1] = new(adapters.PBSOrtbBid)
	sb1.Bids[0].Bid = new(openrtb.Bid)
	sb1.Bids[1].Bid = new(openrtb.Bid)
	sb1.Bids[0].Bid.ID = "1234567890"
	sb1.Bids[1].Bid.ID = "5678901234"

	a.seatBid = sb1
	a.delay = 10 * time.Millisecond
}

func mockAdapterConfig2(a *mockAdapter) {
	sb1 := new(adapters.PBSOrtbSeatBid)
	sb1.Bids = make([]*adapters.PBSOrtbBid, 2)
	sb1.Bids[0] = new(adapters.PBSOrtbBid)
	sb1.Bids[1] = new(adapters.PBSOrtbBid)
	sb1.Bids[0].Bid = new(openrtb.Bid)
	sb1.Bids[1].Bid = new(openrtb.Bid)
	sb1.Bids[0].Bid.ID = "ABC"
	sb1.Bids[1].Bid.ID = "1234"

	a.seatBid = sb1
	a.delay = 5 * time.Millisecond
}

func mockAdapterConfig3(a *mockAdapter) {
	sb1 := new(adapters.PBSOrtbSeatBid)
	sb1.Bids = make([]*adapters.PBSOrtbBid, 1)
	sb1.Bids[0] = new(adapters.PBSOrtbBid)
	sb1.Bids[0].Bid = new(openrtb.Bid)
	sb1.Bids[0].Bid.ID = "MyBid"

	a.seatBid = sb1
	a.delay = 2 * time.Millisecond
}

func mockAdapterConfigErr(a *mockAdapter) {
	sb1 := new(adapters.PBSOrtbSeatBid)

	a.seatBid = sb1
	a.delay = 2 * time.Millisecond
	a.errs = append(a.errs, &errorString{ "This was an error" })

}

// errorString is a trivial implementation of error.
type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}