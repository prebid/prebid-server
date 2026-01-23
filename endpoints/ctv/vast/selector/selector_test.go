package selector

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestSelector_Single(t *testing.T) {
	response := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", Price: 5.0, ImpID: "imp1"},
					{ID: "bid2", Price: 3.0, ImpID: "imp1"},
				},
			},
			{
				Seat: "bidder2",
				Bid: []openrtb2.Bid{
					{ID: "bid3", Price: 7.0, ImpID: "imp1"},
				},
			},
		},
	}

	selector := NewSelector()
	config := Config{
		Strategy:    StrategySingle,
		MaxAdsInPod: 1,
	}

	result, err := selector.Select(response, config)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}

	if len(result.Bids) != 1 {
		t.Fatalf("Expected 1 bid, got %d", len(result.Bids))
	}

	// Should select bid3 with highest price (7.0)
	if result.Bids[0].Bid.ID != "bid3" {
		t.Errorf("Expected bid3, got %s", result.Bids[0].Bid.ID)
	}

	if result.Bids[0].Bid.Price != 7.0 {
		t.Errorf("Expected price 7.0, got %f", result.Bids[0].Bid.Price)
	}

	if result.Bids[0].Seat != "bidder2" {
		t.Errorf("Expected seat bidder2, got %s", result.Bids[0].Seat)
	}

	// Check rejected bids
	if len(result.Rejected) != 2 {
		t.Errorf("Expected 2 rejected bids, got %d", len(result.Rejected))
	}
}

func TestSelector_TopN(t *testing.T) {
	response := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", Price: 5.0, ImpID: "imp1"},
					{ID: "bid2", Price: 3.0, ImpID: "imp1"},
				},
			},
			{
				Seat: "bidder2",
				Bid: []openrtb2.Bid{
					{ID: "bid3", Price: 7.0, ImpID: "imp1"},
					{ID: "bid4", Price: 2.0, ImpID: "imp1"},
				},
			},
		},
	}

	selector := NewSelector()
	config := Config{
		Strategy:    StrategyTopN,
		MaxAdsInPod: 3,
	}

	result, err := selector.Select(response, config)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}

	if len(result.Bids) != 3 {
		t.Fatalf("Expected 3 bids, got %d", len(result.Bids))
	}

	// Should be sorted by price: bid3 (7.0), bid1 (5.0), bid2 (3.0)
	expectedIDs := []string{"bid3", "bid1", "bid2"}
	for i, expected := range expectedIDs {
		if result.Bids[i].Bid.ID != expected {
			t.Errorf("Position %d: expected %s, got %s", i, expected, result.Bids[i].Bid.ID)
		}
	}

	// Check rejected bid
	if len(result.Rejected) != 1 {
		t.Fatalf("Expected 1 rejected bid, got %d", len(result.Rejected))
	}

	if result.Rejected[0].Bid.Bid.ID != "bid4" {
		t.Errorf("Expected rejected bid4, got %s", result.Rejected[0].Bid.Bid.ID)
	}

	if result.Rejected[0].Reason != "exceeded_max_ads_in_pod" {
		t.Errorf("Expected reason 'exceeded_max_ads_in_pod', got '%s'", result.Rejected[0].Reason)
	}
}

func TestSelector_EmptyResponse(t *testing.T) {
	response := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{},
	}

	selector := NewSelector()
	config := Config{
		Strategy:    StrategySingle,
		MaxAdsInPod: 1,
	}

	result, err := selector.Select(response, config)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}

	if len(result.Bids) != 0 {
		t.Errorf("Expected 0 bids, got %d", len(result.Bids))
	}

	if len(result.Rejected) != 0 {
		t.Errorf("Expected 0 rejected bids, got %d", len(result.Rejected))
	}
}

func TestSelector_NilResponse(t *testing.T) {
	selector := NewSelector()
	config := Config{
		Strategy:    StrategySingle,
		MaxAdsInPod: 1,
	}

	_, err := selector.Select(nil, config)
	if err == nil {
		t.Error("Expected error for nil response")
	}
}

func TestSelector_TopN_MaxGreaterThanBids(t *testing.T) {
	response := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", Price: 5.0, ImpID: "imp1"},
					{ID: "bid2", Price: 3.0, ImpID: "imp1"},
				},
			},
		},
	}

	selector := NewSelector()
	config := Config{
		Strategy:    StrategyTopN,
		MaxAdsInPod: 10, // More than available bids
	}

	result, err := selector.Select(response, config)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}

	// Should return all 2 bids
	if len(result.Bids) != 2 {
		t.Errorf("Expected 2 bids, got %d", len(result.Bids))
	}

	// No rejected bids
	if len(result.Rejected) != 0 {
		t.Errorf("Expected 0 rejected bids, got %d", len(result.Rejected))
	}
}

func TestSelector_SamePriceSorting(t *testing.T) {
	response := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid-z", Price: 5.0, ImpID: "imp1"},
					{ID: "bid-a", Price: 5.0, ImpID: "imp1"},
					{ID: "bid-m", Price: 5.0, ImpID: "imp1"},
				},
			},
		},
	}

	selector := NewSelector()
	config := Config{
		Strategy:    StrategyTopN,
		MaxAdsInPod: 3,
	}

	result, err := selector.Select(response, config)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}

	// Should be sorted alphabetically by ID when prices are same
	expectedIDs := []string{"bid-a", "bid-m", "bid-z"}
	for i, expected := range expectedIDs {
		if result.Bids[i].Bid.ID != expected {
			t.Errorf("Position %d: expected %s, got %s", i, expected, result.Bids[i].Bid.ID)
		}
	}
}

func TestSelector_DefaultMaxAdsInPod(t *testing.T) {
	response := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", Price: 5.0},
					{ID: "bid2", Price: 3.0},
				},
			},
		},
	}

	selector := NewSelector()
	config := Config{
		Strategy:    StrategySingle,
		MaxAdsInPod: 0, // Should default to 1
	}

	result, err := selector.Select(response, config)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}

	if len(result.Bids) != 1 {
		t.Errorf("Expected 1 bid when MaxAdsInPod is 0, got %d", len(result.Bids))
	}
}

func TestGetSequence(t *testing.T) {
	tests := []struct {
		name     string
		index    int
		expected int
	}{
		{"First position", 0, 1},
		{"Second position", 1, 2},
		{"Third position", 2, 3},
		{"Negative index", -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bid := &openrtb2.Bid{ID: "test"}
			result := GetSequence(bid, tt.index)
			if result != tt.expected {
				t.Errorf("Expected sequence %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestSelector_UnknownStrategy(t *testing.T) {
	response := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", Price: 5.0},
					{ID: "bid2", Price: 3.0},
				},
			},
		},
	}

	selector := NewSelector()
	config := Config{
		Strategy:    "UNKNOWN",
		MaxAdsInPod: 1,
	}

	result, err := selector.Select(response, config)
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}

	// Should default to single selection
	if len(result.Bids) != 1 {
		t.Errorf("Expected 1 bid for unknown strategy, got %d", len(result.Bids))
	}
}
