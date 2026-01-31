package selector

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/ctv/vast/core"
)

func TestPriceSelector_Select_NilResponse(t *testing.T) {
	selector := &PriceSelector{}
	cfg := core.ReceiverConfig{
		SelectionStrategy: "SINGLE",
	}

	selected, warnings, err := selector.Select(nil, nil, cfg)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(selected) != 0 {
		t.Errorf("Expected 0 selected bids, got %d", len(selected))
	}
	if len(warnings) == 0 {
		t.Error("Expected warnings for nil response")
	}
}

func TestPriceSelector_Select_EmptyResponse(t *testing.T) {
	selector := &PriceSelector{}
	cfg := core.ReceiverConfig{
		SelectionStrategy: "SINGLE",
	}

	resp := &openrtb2.BidResponse{
		ID:      "test",
		SeatBid: []openrtb2.SeatBid{},
	}

	selected, warnings, err := selector.Select(nil, resp, cfg)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(selected) != 0 {
		t.Errorf("Expected 0 selected bids, got %d", len(selected))
	}
	if len(warnings) == 0 {
		t.Error("Expected warnings for empty response")
	}
}

func TestPriceSelector_Select_SingleStrategy(t *testing.T) {
	selector := &PriceSelector{}
	cfg := core.ReceiverConfig{
		SelectionStrategy: "SINGLE",
		DefaultCurrency:   "USD",
	}

	resp := &openrtb2.BidResponse{
		ID:  "test",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", ImpID: "imp1", Price: 5.0, AdM: "<VAST/>"},
					{ID: "bid2", ImpID: "imp1", Price: 3.0, AdM: "<VAST/>"},
					{ID: "bid3", ImpID: "imp1", Price: 7.0, AdM: "<VAST/>"},
				},
			},
		},
	}

	selected, warnings, err := selector.Select(nil, resp, cfg)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(selected) != 1 {
		t.Fatalf("Expected 1 selected bid, got %d", len(selected))
	}

	// Should select highest price bid (bid3 with price 7.0)
	if selected[0].Bid.ID != "bid3" {
		t.Errorf("Expected bid3, got %s", selected[0].Bid.ID)
	}
	if selected[0].Bid.Price != 7.0 {
		t.Errorf("Expected price 7.0, got %f", selected[0].Bid.Price)
	}
	if selected[0].Sequence != 1 {
		t.Errorf("Expected sequence 1, got %d", selected[0].Sequence)
	}
	if selected[0].Seat != "bidder1" {
		t.Errorf("Expected seat bidder1, got %s", selected[0].Seat)
	}

	// Check metadata
	if selected[0].Meta.BidID != "bid3" {
		t.Errorf("Expected meta.BidID bid3, got %s", selected[0].Meta.BidID)
	}
	if selected[0].Meta.Price != 7.0 {
		t.Errorf("Expected meta.Price 7.0, got %f", selected[0].Meta.Price)
	}
	if selected[0].Meta.Currency != "USD" {
		t.Errorf("Expected meta.Currency USD, got %s", selected[0].Meta.Currency)
	}

	if len(warnings) > 0 {
		t.Logf("Warnings: %v", warnings)
	}
}

func TestPriceSelector_Select_TopNStrategy(t *testing.T) {
	selector := &PriceSelector{}
	cfg := core.ReceiverConfig{
		SelectionStrategy: "TOP_N",
		MaxAdsInPod:       3,
		DefaultCurrency:   "USD",
	}

	resp := &openrtb2.BidResponse{
		ID:  "test",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", ImpID: "imp1", Price: 5.0, AdM: "<VAST/>"},
					{ID: "bid2", ImpID: "imp1", Price: 3.0, AdM: "<VAST/>"},
					{ID: "bid3", ImpID: "imp1", Price: 7.0, AdM: "<VAST/>"},
					{ID: "bid4", ImpID: "imp1", Price: 6.0, AdM: "<VAST/>"},
					{ID: "bid5", ImpID: "imp1", Price: 2.0, AdM: "<VAST/>"},
				},
			},
		},
	}

	selected, warnings, err := selector.Select(nil, resp, cfg)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(selected) != 3 {
		t.Fatalf("Expected 3 selected bids, got %d", len(selected))
	}

	// Should select top 3 by price: bid3 (7.0), bid4 (6.0), bid1 (5.0)
	expectedIDs := []string{"bid3", "bid4", "bid1"}
	expectedPrices := []float64{7.0, 6.0, 5.0}
	expectedSequences := []int{1, 2, 3}

	for i, exp := range expectedIDs {
		if selected[i].Bid.ID != exp {
			t.Errorf("Position %d: expected %s, got %s", i, exp, selected[i].Bid.ID)
		}
		if selected[i].Bid.Price != expectedPrices[i] {
			t.Errorf("Position %d: expected price %f, got %f", i, expectedPrices[i], selected[i].Bid.Price)
		}
		if selected[i].Sequence != expectedSequences[i] {
			t.Errorf("Position %d: expected sequence %d, got %d", i, expectedSequences[i], selected[i].Sequence)
		}
	}

	// Should have warning about limiting
	hasLimitWarning := false
	for _, w := range warnings {
		if len(w) > 0 {
			hasLimitWarning = true
		}
	}
	if !hasLimitWarning {
		t.Error("Expected warning about limiting to MaxAdsInPod")
	}
}

func TestPriceSelector_FilterBids_Price(t *testing.T) {
	selector := &PriceSelector{}
	cfg := core.ReceiverConfig{
		SelectionStrategy: "SINGLE",
		DefaultCurrency:   "USD",
	}

	resp := &openrtb2.BidResponse{
		ID:  "test",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", ImpID: "imp1", Price: 0.0, AdM: "<VAST/>"},   // Invalid: price = 0
					{ID: "bid2", ImpID: "imp1", Price: -1.0, AdM: "<VAST/>"},  // Invalid: price < 0
					{ID: "bid3", ImpID: "imp1", Price: 5.0, AdM: "<VAST/>"},   // Valid
				},
			},
		},
	}

	selected, warnings, err := selector.Select(nil, resp, cfg)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(selected) != 1 {
		t.Fatalf("Expected 1 selected bid, got %d", len(selected))
	}
	if selected[0].Bid.ID != "bid3" {
		t.Errorf("Expected bid3, got %s", selected[0].Bid.ID)
	}

	// Should have warnings about filtered bids
	if len(warnings) < 2 {
		t.Errorf("Expected at least 2 warnings about filtered bids, got %d", len(warnings))
	}
}

func TestPriceSelector_FilterBids_AdM(t *testing.T) {
	selector := &PriceSelector{}
	cfg := core.ReceiverConfig{
		SelectionStrategy:  "SINGLE",
		DefaultCurrency:    "USD",
		AllowSkeletonVast: false,
	}

	resp := &openrtb2.BidResponse{
		ID:  "test",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", ImpID: "imp1", Price: 5.0, AdM: ""},         // Invalid: empty adm
					{ID: "bid2", ImpID: "imp1", Price: 3.0, AdM: "<VAST/>"},  // Valid
				},
			},
		},
	}

	selected, _, err := selector.Select(nil, resp, cfg)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(selected) != 1 {
		t.Fatalf("Expected 1 selected bid, got %d", len(selected))
	}
	if selected[0].Bid.ID != "bid2" {
		t.Errorf("Expected bid2, got %s", selected[0].Bid.ID)
	}
}

func TestPriceSelector_FilterBids_AllowSkeletonVast(t *testing.T) {
	selector := &PriceSelector{}
	cfg := core.ReceiverConfig{
		SelectionStrategy:  "TOP_N",
		MaxAdsInPod:        5,
		DefaultCurrency:    "USD",
		AllowSkeletonVast: true,
	}

	resp := &openrtb2.BidResponse{
		ID:  "test",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", ImpID: "imp1", Price: 5.0, AdM: ""},         // Valid with AllowSkeletonVast
					{ID: "bid2", ImpID: "imp1", Price: 3.0, AdM: "<VAST/>"},  // Valid
				},
			},
		},
	}

	selected, _, err := selector.Select(nil, resp, cfg)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(selected) != 2 {
		t.Fatalf("Expected 2 selected bids, got %d", len(selected))
	}
}

func TestPriceSelector_SortBids_DealPriority(t *testing.T) {
	selector := &PriceSelector{}
	cfg := core.ReceiverConfig{
		SelectionStrategy: "TOP_N",
		MaxAdsInPod:       3,
		DefaultCurrency:   "USD",
	}

	resp := &openrtb2.BidResponse{
		ID:  "test",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", ImpID: "imp1", Price: 5.0, DealID: "", AdM: "<VAST/>"},       // No deal
					{ID: "bid2", ImpID: "imp1", Price: 5.0, DealID: "deal1", AdM: "<VAST/>"},  // Same price, has deal
					{ID: "bid3", ImpID: "imp1", Price: 5.0, DealID: "", AdM: "<VAST/>"},       // No deal
				},
			},
		},
	}

	selected, warnings, err := selector.Select(nil, resp, cfg)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(selected) != 3 {
		t.Fatalf("Expected 3 selected bids, got %d", len(selected))
	}

	// bid2 should be first (same price but has deal)
	if selected[0].Bid.ID != "bid2" {
		t.Errorf("Expected bid2 first (has deal), got %s", selected[0].Bid.ID)
	}

	t.Logf("Selected order: %s, %s, %s", selected[0].Bid.ID, selected[1].Bid.ID, selected[2].Bid.ID)
	t.Logf("Warnings: %v", warnings)
}

func TestPriceSelector_SortBids_IdStability(t *testing.T) {
	selector := &PriceSelector{}
	cfg := core.ReceiverConfig{
		SelectionStrategy: "TOP_N",
		MaxAdsInPod:       3,
		DefaultCurrency:   "USD",
	}

	resp := &openrtb2.BidResponse{
		ID:  "test",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid_c", ImpID: "imp1", Price: 5.0, AdM: "<VAST/>"},
					{ID: "bid_a", ImpID: "imp1", Price: 5.0, AdM: "<VAST/>"},
					{ID: "bid_b", ImpID: "imp1", Price: 5.0, AdM: "<VAST/>"},
				},
			},
		},
	}

	selected, _, err := selector.Select(nil, resp, cfg)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(selected) != 3 {
		t.Fatalf("Expected 3 selected bids, got %d", len(selected))
	}

	// Should be sorted by ID alphabetically: bid_a, bid_b, bid_c
	expectedOrder := []string{"bid_a", "bid_b", "bid_c"}
	for i, exp := range expectedOrder {
		if selected[i].Bid.ID != exp {
			t.Errorf("Position %d: expected %s, got %s", i, exp, selected[i].Bid.ID)
		}
	}
}

func TestPriceSelector_ExtractMetadata(t *testing.T) {
	selector := &PriceSelector{}

	dur30 := int64(30)

	bid := &openrtb2.Bid{
		ID:      "bid1",
		ImpID:   "imp1",
		Price:   5.50,
		ADomain: []string{"example.com", "other.com"},
		Cat:     []string{"IAB1-1", "IAB2-2"},
		DealID:  "deal123",
		W:       640,
		Dur:     dur30,
	}

	meta := selector.extractMetadata(bid, "bidder1", "EUR", 2)

	if meta.BidID != "bid1" {
		t.Errorf("Expected BidID bid1, got %s", meta.BidID)
	}
	if meta.ImpID != "imp1" {
		t.Errorf("Expected ImpID imp1, got %s", meta.ImpID)
	}
	if meta.Price != 5.50 {
		t.Errorf("Expected Price 5.50, got %f", meta.Price)
	}
	if meta.Currency != "EUR" {
		t.Errorf("Expected Currency EUR, got %s", meta.Currency)
	}
	if meta.Seat != "bidder1" {
		t.Errorf("Expected Seat bidder1, got %s", meta.Seat)
	}
	if meta.Adomain != "example.com" {
		t.Errorf("Expected Adomain example.com, got %s", meta.Adomain)
	}
	if len(meta.Cats) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(meta.Cats))
	}
	if meta.DealID != "deal123" {
		t.Errorf("Expected DealID deal123, got %s", meta.DealID)
	}
	if meta.DurSec != 30 {
		t.Errorf("Expected DurSec 30, got %d", meta.DurSec)
	}
	if meta.SlotInPod != 2 {
		t.Errorf("Expected SlotInPod 2, got %d", meta.SlotInPod)
	}
}

func TestPriceSelector_GetCurrency(t *testing.T) {
	selector := &PriceSelector{}

	// Test with response currency
	resp := &openrtb2.BidResponse{
		Cur: "EUR",
	}
	cfg := core.ReceiverConfig{
		DefaultCurrency: "USD",
	}

	currency := selector.getCurrency(resp, cfg)
	if currency != "EUR" {
		t.Errorf("Expected EUR from response, got %s", currency)
	}

	// Test with empty response currency (should use default)
	resp.Cur = ""
	currency = selector.getCurrency(resp, cfg)
	if currency != "USD" {
		t.Errorf("Expected USD from config, got %s", currency)
	}
}

func TestPriceSelector_MultipleSeatBids(t *testing.T) {
	selector := &PriceSelector{}
	cfg := core.ReceiverConfig{
		SelectionStrategy: "TOP_N",
		MaxAdsInPod:       4,
		DefaultCurrency:   "USD",
	}

	resp := &openrtb2.BidResponse{
		ID:  "test",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", ImpID: "imp1", Price: 5.0, AdM: "<VAST/>"},
					{ID: "bid2", ImpID: "imp1", Price: 3.0, AdM: "<VAST/>"},
				},
			},
			{
				Seat: "bidder2",
				Bid: []openrtb2.Bid{
					{ID: "bid3", ImpID: "imp1", Price: 7.0, AdM: "<VAST/>"},
					{ID: "bid4", ImpID: "imp1", Price: 4.0, AdM: "<VAST/>"},
				},
			},
		},
	}

	selected, _, err := selector.Select(nil, resp, cfg)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(selected) != 4 {
		t.Fatalf("Expected 4 selected bids, got %d", len(selected))
	}

	// Should select all 4 bids sorted by price: bid3 (7.0), bid1 (5.0), bid4 (4.0), bid2 (3.0)
	expectedIDs := []string{"bid3", "bid1", "bid4", "bid2"}
	expectedSeats := []string{"bidder2", "bidder1", "bidder2", "bidder1"}

	for i := range expectedIDs {
		if selected[i].Bid.ID != expectedIDs[i] {
			t.Errorf("Position %d: expected %s, got %s", i, expectedIDs[i], selected[i].Bid.ID)
		}
		if selected[i].Seat != expectedSeats[i] {
			t.Errorf("Position %d: expected seat %s, got %s", i, expectedSeats[i], selected[i].Seat)
		}
	}
}
