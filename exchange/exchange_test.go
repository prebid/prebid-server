package exchange

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"github.com/mxmCherry/openrtb"
	"context"
	"github.com/prebid/prebid-server/openrtb_ext"
	"time"
	"encoding/json"
	"errors"
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

	e := NewDummyExchange(server.Client())
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

	bidResponse, err := e.HoldAuction(ctx, bidRequest)
	if err != nil {
		t.Errorf("HoldAuction: %s", err.Error())
	}
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

	e := NewDummyExchange(server.Client())
	mockAdapterConfig1(e.adapterMap[BidderDummy].(*mockAdapter))
	mockAdapterConfig2(e.adapterMap[BidderDummy2].(*mockAdapter))
	mockAdapterConfig3(e.adapterMap[BidderDummy3].(*mockAdapter))

	cleanRequests := make(map[openrtb_ext.BidderName]*openrtb.BidRequest)
	adapterBids, adapterExtra := e.getAllBids(ctx, e.adapters, cleanRequests)

	if len(adapterBids[BidderDummy].bids) != 2 {
		t.Errorf("GetAllBids failed to get 2 bids from BidderDummy, found %d instead", len(adapterBids[BidderDummy].bids))
	}
	if adapterBids[BidderDummy].bids[0].bid.ID != "1234567890" {
		t.Errorf("GetAllBids failed to get the first bid of BidderDummy")
	}
	if adapterBids[BidderDummy3].bids[0].bid.ID != "MyBid" {
		t.Errorf("GetAllBids failed to get the bid from BidderDummy3")
	}
	if len(adapterExtra) != 3 {
		t.Errorf("GetAllBids failed to return 3 adapterExtra's, got %d instead", len(adapterExtra))
	}

	// Now test with an error condition
	mockAdapterConfigErr1(e.adapterMap[BidderDummy2].(*mockAdapter))
	if len(e.adapterMap[BidderDummy2].(*mockAdapter).errs) != 2 {
		t.Errorf("GetAllBids, Bidder2 adapter error generation failed. Only seeing %d errors", len(e.adapterMap[BidderDummy2].(*mockAdapter).errs))
	}
	adapterBids, adapterExtra = e.getAllBids(ctx, e.adapters, cleanRequests)

	if len(e.adapterMap[BidderDummy2].(*mockAdapter).errs) != 2 {
		t.Errorf("GetAllBids, Bidder2 adapter error generation failed. Only seeing %d errors", len(e.adapterMap[BidderDummy2].(*mockAdapter).errs))
	}
	if len(adapterExtra[BidderDummy2].Errors) !=2 {
		t.Errorf("GetAllBids failed to report 2 errors on Bidder2, found %d errors", len(adapterExtra[BidderDummy2].Errors))
	}
	if len(adapterExtra[BidderDummy].Errors) !=0 {
		t.Errorf("GetAllBids found errors on Bidder1, found %d errors", len(adapterExtra[BidderDummy2].Errors))
	}
	if len(adapterBids[BidderDummy2].bids) !=0 {
		t.Errorf("GetAllBids found bids on Bidder2, found %d bids", len(adapterBids[BidderDummy2].bids))
	}

	// Test with null pointer for bid response
	mockAdapterConfigErr2(e.adapterMap[BidderDummy2].(*mockAdapter))
	adapterBids, adapterExtra = e.getAllBids(ctx, e.adapters, cleanRequests)

	if len(adapterExtra[BidderDummy2].Errors) !=1 {
		t.Errorf("GetAllBids failed to report 1 errors on Bidder2, found %d errors", len(adapterExtra[BidderDummy2].Errors))
	}
	if len(adapterExtra[BidderDummy].Errors) !=0 {
		t.Errorf("GetAllBids found errors on Bidder1, found %d errors", len(adapterExtra[BidderDummy2].Errors))
	}

}

func TestBuildBidResponse(t *testing.T) {
	//  BuildBidResponse(liveAdapters []openrtb_ext.BidderName, adapterBids map[openrtb_ext.BidderName]*adapters.pbsOrtbSeatBid, bidRequest *openrtb.BidRequest, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra) *openrtb.BidResponse
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	e := NewDummyExchange(server.Client())
	mockAdapterConfig1(e.adapterMap[BidderDummy].(*mockAdapter))
	mockAdapterConfig2(e.adapterMap[BidderDummy2].(*mockAdapter))
	mockAdapterConfig3(e.adapterMap[BidderDummy3].(*mockAdapter))

	// Very simple Bid request. At this point we are just reading these two values
	bidRequest := openrtb.BidRequest{
		ID: "This Bid",
		Test: 0,
	}

	liveAdapters := make([]openrtb_ext.BidderName, 3)
	liveAdapters[0] = BidderDummy
	liveAdapters[1] = BidderDummy2
	liveAdapters[2] = BidderDummy3

	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)
	adapterExtra := make(map[openrtb_ext.BidderName]*seatResponseExtra)

	var errs1, errs2, errs3 []error
	adapterBids[BidderDummy], errs1 = mockDummyBids1()
	adapterBids[BidderDummy2], errs2 = mockDummyBids2()
	adapterBids[BidderDummy3], errs3 = mockDummyBids3()
	adapterExtra[BidderDummy] = &seatResponseExtra{ResponseTimeMillis: 131, Errors:convertErr2Str(errs1)}
	adapterExtra[BidderDummy2] = &seatResponseExtra{ResponseTimeMillis: 97, Errors:convertErr2Str(errs2)}
	adapterExtra[BidderDummy3] = &seatResponseExtra{ResponseTimeMillis: 141, Errors:convertErr2Str(errs3)}

	errList := make([]error, 0, 1)
	bidResponse, err := e.buildBidResponse(liveAdapters, adapterBids, &bidRequest, adapterExtra, errList)
	if err != nil {
		t.Errorf("BuildBidResponse: %s", err.Error())
	}
	bidResponseExt := new(openrtb_ext.ExtBidResponse)
	_ = json.Unmarshal(bidResponse.Ext, bidResponseExt)

	if len(bidResponseExt.ResponseTimeMillis) != 3 {
		t.Errorf("BuildBidResponse: Did not find 3 response times. Found %d instead", len(bidResponseExt.ResponseTimeMillis))
	}
	if len(bidResponse.SeatBid) != 3 {
		t.Errorf("BuildBidResponse: Expected 3 SeatBids, found %d instead", len(bidResponse.SeatBid))
	}
	if len(bidResponse.SeatBid) != 3 {
		t.Errorf("BuildBidResponse: Expected 3 SeatBids, found %d instead", len(bidResponse.SeatBid))
	}

	// Now test with an error condition
	adapterBids[BidderDummy2], errs2 = mockDummyBidsErr1()
	adapterExtra[BidderDummy2] = &seatResponseExtra{ResponseTimeMillis: 97, Errors:convertErr2Str(errs2)}

	bidResponse, err = e.buildBidResponse(liveAdapters, adapterBids, &bidRequest, adapterExtra, errList)
	if err != nil {
		t.Errorf("BuildBidResponse: %s", err.Error())
	}
	bidResponseExt = new(openrtb_ext.ExtBidResponse)
	_ = json.Unmarshal(bidResponse.Ext, bidResponseExt)

	// This case we know the order of the adapters, as GetAllBids have not scrambled them
	if len(bidResponse.SeatBid[0].Bid) != 2 {
		t.Errorf("BuildBidResponse: Bidder 1 expected 2 bids, found %d", len(bidResponse.SeatBid[0].Bid))
	}
	if bidResponse.SeatBid[1].Bid[0].ID != "MyBid" {
		t.Errorf("BuildBidResponse: Bidder 3 bid ID not correct. Expected \"MyBid\", found \"%s\"", bidResponse.SeatBid[2].Bid[0].ID)
	}

	// Test with null bid response error
	adapterBids[BidderDummy2], errs2 = mockDummyBidsErr2()
	adapterExtra[BidderDummy2] = &seatResponseExtra{ResponseTimeMillis: 97, Errors:convertErr2Str(errs2)}

	bidResponse, err = e.buildBidResponse(liveAdapters, adapterBids, &bidRequest, adapterExtra, errList)
	if err != nil {
		t.Errorf("BuildBidResponse: %s", err.Error())
	}
	bidResponseExt = new(openrtb_ext.ExtBidResponse)
	_ = json.Unmarshal(bidResponse.Ext, bidResponseExt)

	// This case we know the order of the adapters, as GetAllBids have not scrambled them
	if len(bidResponse.SeatBid[0].Bid) != 2 {
		t.Errorf("BuildBidResponse: Bidder 1 expected 2 bids, found %d", len(bidResponse.SeatBid[0].Bid))
	}
	if bidResponse.SeatBid[1].Bid[0].ID != "MyBid" {
		t.Errorf("BuildBidResponse: Bidder 3 bid ID not correct. Expected \"MyBid\", found \"%s\"", bidResponse.SeatBid[2].Bid[0].ID)
	}

}

type mockAdapter struct {
	seatBid *pbsOrtbSeatBid
	errs []error
	delay time.Duration
}

func (a *mockAdapter) requestBid(ctx context.Context, request *openrtb.BidRequest) (*pbsOrtbSeatBid, []error) {
	time.Sleep(a.delay)
	return a.seatBid, a.errs
}

const (
	BidderDummy openrtb_ext.BidderName = "dummy"
	BidderDummy2 openrtb_ext.BidderName = "dummy2"
	BidderDummy3 openrtb_ext.BidderName = "dummy3"
)
// Tester is responsible for filling bid results into the adapters
func NewDummyExchange(client *http.Client) *exchange {
	e := new(exchange)
	a := new(mockAdapter)
	a.errs = make([]error, 0, 5)

	b := new(mockAdapter)
	b.errs = make([]error, 0, 5)

	c := new(mockAdapter)
	c.errs = make([]error, 0, 5)

	e.adapterMap = map[openrtb_ext.BidderName]adaptedBidder{
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
	a.seatBid, a.errs = mockDummyBids1()

	a.delay = 0 * time.Microsecond
}

func mockAdapterConfig2(a *mockAdapter) {
	a.seatBid, a.errs = mockDummyBids2()

	a.delay = 0 * time.Microsecond
}

func mockAdapterConfig3(a *mockAdapter) {
	a.seatBid, a.errs = mockDummyBids3()

	a.delay = 0 * time.Microsecond
}

func mockAdapterConfigErr1(a *mockAdapter) {
	a.seatBid, a.errs = mockDummyBidsErr1()

	a.delay = 0 * time.Microsecond
}

func mockAdapterConfigErr2(a *mockAdapter) {
	a.seatBid, a.errs = mockDummyBidsErr2()

	a.delay = 0 * time.Microsecond
}

func mockDummyBids1() (*pbsOrtbSeatBid, []error) {
	sb1 := new(pbsOrtbSeatBid)
	sb1.bids = make([]*pbsOrtbBid, 2)
	sb1.bids[0] = new(pbsOrtbBid)
	sb1.bids[1] = new(pbsOrtbBid)
	sb1.bids[0].bid = new(openrtb.Bid)
	sb1.bids[1].bid = new(openrtb.Bid)
	sb1.bids[0].bid.ID = "1234567890"
	sb1.bids[1].bid.ID = "5678901234"

	errs := make([]error, 0, 5)

	return sb1, errs
}

func mockDummyBids2() (*pbsOrtbSeatBid, []error) {
	sb1 := new(pbsOrtbSeatBid)
	sb1.bids = make([]*pbsOrtbBid, 2)
	sb1.bids[0] = new(pbsOrtbBid)
	sb1.bids[1] = new(pbsOrtbBid)
	sb1.bids[0].bid = new(openrtb.Bid)
	sb1.bids[1].bid = new(openrtb.Bid)
	sb1.bids[0].bid.ID = "ABC"
	sb1.bids[1].bid.ID = "1234"

	errs := make([]error, 0, 5)

	return sb1, errs
}
func mockDummyBids3() (*pbsOrtbSeatBid, []error) {
	sb1 := new(pbsOrtbSeatBid)
	sb1.bids = make([]*pbsOrtbBid, 1)
	sb1.bids[0] = new(pbsOrtbBid)
	sb1.bids[0].bid = new(openrtb.Bid)
	sb1.bids[0].bid.ID = "MyBid"

	errs := make([]error, 0, 5)

	return sb1, errs
}
func mockDummyBidsErr1() (*pbsOrtbSeatBid, []error) {
	sb1 := new(pbsOrtbSeatBid)
	sb1.bids = nil

	errs := make([]error, 0, 5)
	errs = append(errs, errors.New("This was an error") )
	errs = append(errs, errors.New("Another error goes here") )

	return sb1, errs
}
func mockDummyBidsErr2() (*pbsOrtbSeatBid, []error) {
	var sb1 *pbsOrtbSeatBid = nil

	errs := make([]error, 0, 5)
	errs = append(errs, errors.New("This was a FATAL error") )

	return sb1, errs
}


func convertErr2Str(e []error) []string {
	s := make([]string, len(e))
	for i :=0; i<len(e); i++ {
		s[i] = e[i].Error()
	}
	return s
}
