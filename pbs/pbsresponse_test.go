package pbs

import (
	"sort"
	"testing"
)

func TestSortBids(t *testing.T) {
	bid1 := PBSBid{
		BidID:      "testBidId",
		AdUnitCode: "testAdUnitCode",
		BidderCode: "testBidderCode",
		Price:      0.0,
	}
	bid2 := PBSBid{
		BidID:      "testBidId",
		AdUnitCode: "testAdUnitCode",
		BidderCode: "testBidderCode",
		Price:      4.0,
	}
	bid3 := PBSBid{
		BidID:      "testBidId",
		AdUnitCode: "testAdUnitCode",
		BidderCode: "testBidderCode",
		Price:      2.0,
	}
	bid4 := PBSBid{
		BidID:      "testBidId",
		AdUnitCode: "testAdUnitCode",
		BidderCode: "testBidderCode",
		Price:      0.50,
	}

	bids := make(PBSBidSlice, 0)
	bids = append(bids, &bid1, &bid2, &bid3, &bid4)

	sort.Sort(bids)
	if bids[0].Price != 4.0 {
		t.Error("Expected 4.00 to be highest price")
	}
	if bids[1].Price != 2.0 {
		t.Error("Expected 2.00 to be second highest price")
	}
	if bids[2].Price != 0.5 {
		t.Error("Expected 0.50 to be third highest price")
	}
	if bids[3].Price != 0.0 {
		t.Error("Expected 0.00 to be lowest price")
	}
}

func TestSortBidsWithResponseTimes(t *testing.T) {
	bid1 := PBSBid{
		BidID:        "testBidId",
		AdUnitCode:   "testAdUnitCode",
		BidderCode:   "testBidderCode",
		Price:        1.0,
		ResponseTime: 70,
	}
	bid2 := PBSBid{
		BidID:        "testBidId",
		AdUnitCode:   "testAdUnitCode",
		BidderCode:   "testBidderCode",
		Price:        1.0,
		ResponseTime: 20,
	}
	bid3 := PBSBid{
		BidID:        "testBidId",
		AdUnitCode:   "testAdUnitCode",
		BidderCode:   "testBidderCode",
		Price:        1.0,
		ResponseTime: 99,
	}

	bids := make(PBSBidSlice, 0)
	bids = append(bids, &bid1, &bid2, &bid3)

	sort.Sort(bids)
	if bids[0] != &bid2 {
		t.Error("Expected bid 2 to win")
	}
	if bids[1] != &bid1 {
		t.Error("Expected bid 1 to be second")
	}
	if bids[2] != &bid3 {
		t.Error("Expected bid 3 to be last")
	}
}

func TestNilGetPrices(t *testing.T) {
	var bids PBSBidSlice = nil
	if bids.ExtractPrices() != nil {
		t.Error("nil bid slices should return nil price slices.")
	}
}

func TestGetPrices(t *testing.T) {
	price1 := 2.0
	price2 := 0.4
	var bids PBSBidSlice = PBSBidSlice{
		&PBSBid{Price: price1},
		&PBSBid{Price: price2},
	}

	bidPrices := bids.ExtractPrices()
	if len(bidPrices) != 2 {
		t.Fatalf("Expected 2 bid prices. Got %d", len(bidPrices))
	}

	if bidPrices[0] != price1 {
		t.Errorf("Mismatched bid[0] prices. Expected %f, Got %f", price1, bidPrices[0])
	}

	if bidPrices[1] != price2 {
		t.Errorf("Mismatched bid[1] prices. Expected %f, Got %f", price2, bidPrices[1])
	}
}
