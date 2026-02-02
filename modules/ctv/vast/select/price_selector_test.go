package bidselect

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/ctv/vast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSelector(t *testing.T) {
	tests := []struct {
		name     string
		strategy vast.SelectionStrategy
		wantMax  int
	}{
		{
			name:     "SINGLE strategy",
			strategy: vast.SelectionSingle,
			wantMax:  1,
		},
		{
			name:     "TOP_N strategy",
			strategy: vast.SelectionTopN,
			wantMax:  0, // uses cfg.MaxAdsInPod
		},
		{
			name:     "unknown strategy defaults to TOP_N",
			strategy: "unknown",
			wantMax:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewSelector(tt.strategy)
			require.NotNil(t, selector)

			priceSelector, ok := selector.(*PriceSelector)
			require.True(t, ok)
			assert.Equal(t, tt.wantMax, priceSelector.maxBids)
		})
	}
}

func TestPriceSelector_Select_NilResponse(t *testing.T) {
	selector := NewPriceSelector(5)
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "USD",
		MaxAdsInPod:     5,
	}

	selected, warnings, err := selector.Select(nil, nil, cfg)
	assert.NoError(t, err)
	assert.Nil(t, selected)
	assert.Empty(t, warnings)
}

func TestPriceSelector_Select_EmptySeatBid(t *testing.T) {
	selector := NewPriceSelector(5)
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "USD",
		MaxAdsInPod:     5,
	}
	resp := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{},
	}

	selected, warnings, err := selector.Select(nil, resp, cfg)
	assert.NoError(t, err)
	assert.Nil(t, selected)
	assert.Empty(t, warnings)
}

func TestPriceSelector_Select_FilterZeroPrice(t *testing.T) {
	selector := NewPriceSelector(5)
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "USD",
		MaxAdsInPod:     5,
	}
	resp := &openrtb2.BidResponse{
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", Price: 0, AdM: "<VAST></VAST>"},
					{ID: "bid2", Price: -1, AdM: "<VAST></VAST>"},
				},
			},
		},
	}

	selected, warnings, err := selector.Select(nil, resp, cfg)
	assert.NoError(t, err)
	assert.Empty(t, selected)
	assert.Len(t, warnings, 2)
	assert.Contains(t, warnings[0], "price <= 0")
}

func TestPriceSelector_Select_FilterEmptyAdM(t *testing.T) {
	selector := NewPriceSelector(5)
	cfg := vast.ReceiverConfig{
		DefaultCurrency:   "USD",
		MaxAdsInPod:       5,
		AllowSkeletonVast: false,
	}
	resp := &openrtb2.BidResponse{
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", Price: 1.0, AdM: ""},
					{ID: "bid2", Price: 2.0, AdM: "   "},
				},
			},
		},
	}

	selected, warnings, err := selector.Select(nil, resp, cfg)
	assert.NoError(t, err)
	assert.Empty(t, selected)
	assert.Len(t, warnings, 2)
	assert.Contains(t, warnings[0], "empty AdM")
}

func TestPriceSelector_Select_AllowSkeletonVast(t *testing.T) {
	selector := NewPriceSelector(5)
	cfg := vast.ReceiverConfig{
		DefaultCurrency:   "USD",
		MaxAdsInPod:       5,
		AllowSkeletonVast: true,
	}
	resp := &openrtb2.BidResponse{
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", Price: 1.0, AdM: ""},
					{ID: "bid2", Price: 2.0, AdM: ""},
				},
			},
		},
	}

	selected, warnings, err := selector.Select(nil, resp, cfg)
	assert.NoError(t, err)
	assert.Len(t, selected, 2)
	assert.Empty(t, warnings)
}

func TestPriceSelector_Select_SortByPriceDesc(t *testing.T) {
	selector := NewPriceSelector(0)
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "USD",
		MaxAdsInPod:     5,
	}
	resp := &openrtb2.BidResponse{
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", Price: 1.0, AdM: "<VAST></VAST>"},
					{ID: "bid2", Price: 3.0, AdM: "<VAST></VAST>"},
					{ID: "bid3", Price: 2.0, AdM: "<VAST></VAST>"},
				},
			},
		},
	}

	selected, _, err := selector.Select(nil, resp, cfg)
	assert.NoError(t, err)
	require.Len(t, selected, 3)

	// Should be sorted by price descending
	assert.Equal(t, "bid2", selected[0].Meta.BidID)
	assert.Equal(t, 3.0, selected[0].Meta.Price)
	assert.Equal(t, "bid3", selected[1].Meta.BidID)
	assert.Equal(t, 2.0, selected[1].Meta.Price)
	assert.Equal(t, "bid1", selected[2].Meta.BidID)
	assert.Equal(t, 1.0, selected[2].Meta.Price)
}

func TestPriceSelector_Select_DealsPrioritized(t *testing.T) {
	selector := NewPriceSelector(0)
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "USD",
		MaxAdsInPod:     5,
	}
	resp := &openrtb2.BidResponse{
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", Price: 2.0, AdM: "<VAST></VAST>", DealID: ""},
					{ID: "bid2", Price: 2.0, AdM: "<VAST></VAST>", DealID: "deal123"},
				},
			},
		},
	}

	selected, _, err := selector.Select(nil, resp, cfg)
	assert.NoError(t, err)
	require.Len(t, selected, 2)

	// At same price, deal should come first
	assert.Equal(t, "bid2", selected[0].Meta.BidID)
	assert.Equal(t, "deal123", selected[0].Meta.DealID)
	assert.Equal(t, "bid1", selected[1].Meta.BidID)
}

func TestPriceSelector_Select_StableSortByID(t *testing.T) {
	selector := NewPriceSelector(0)
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "USD",
		MaxAdsInPod:     5,
	}
	resp := &openrtb2.BidResponse{
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "c", Price: 2.0, AdM: "<VAST></VAST>"},
					{ID: "a", Price: 2.0, AdM: "<VAST></VAST>"},
					{ID: "b", Price: 2.0, AdM: "<VAST></VAST>"},
				},
			},
		},
	}

	selected, _, err := selector.Select(nil, resp, cfg)
	assert.NoError(t, err)
	require.Len(t, selected, 3)

	// Same price, no deals - should be sorted by ID ascending
	assert.Equal(t, "a", selected[0].Meta.BidID)
	assert.Equal(t, "b", selected[1].Meta.BidID)
	assert.Equal(t, "c", selected[2].Meta.BidID)
}

func TestPriceSelector_Select_SingleStrategy(t *testing.T) {
	selector := NewPriceSelector(1)
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "USD",
		MaxAdsInPod:     5,
	}
	resp := &openrtb2.BidResponse{
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", Price: 1.0, AdM: "<VAST></VAST>"},
					{ID: "bid2", Price: 3.0, AdM: "<VAST></VAST>"},
					{ID: "bid3", Price: 2.0, AdM: "<VAST></VAST>"},
				},
			},
		},
	}

	selected, _, err := selector.Select(nil, resp, cfg)
	assert.NoError(t, err)
	require.Len(t, selected, 1)
	assert.Equal(t, "bid2", selected[0].Meta.BidID)
	assert.Equal(t, 3.0, selected[0].Meta.Price)
}

func TestPriceSelector_Select_TopNRespectsMaxAdsInPod(t *testing.T) {
	selector := NewPriceSelector(0)
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "USD",
		MaxAdsInPod:     2,
	}
	resp := &openrtb2.BidResponse{
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", Price: 1.0, AdM: "<VAST></VAST>"},
					{ID: "bid2", Price: 3.0, AdM: "<VAST></VAST>"},
					{ID: "bid3", Price: 2.0, AdM: "<VAST></VAST>"},
					{ID: "bid4", Price: 4.0, AdM: "<VAST></VAST>"},
				},
			},
		},
	}

	selected, _, err := selector.Select(nil, resp, cfg)
	assert.NoError(t, err)
	require.Len(t, selected, 2)
	assert.Equal(t, "bid4", selected[0].Meta.BidID)
	assert.Equal(t, "bid2", selected[1].Meta.BidID)
}

func TestPriceSelector_Select_Sequence(t *testing.T) {
	selector := NewPriceSelector(0)
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "USD",
		MaxAdsInPod:     5,
	}
	resp := &openrtb2.BidResponse{
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", Price: 1.0, AdM: "<VAST></VAST>"},
					{ID: "bid2", Price: 2.0, AdM: "<VAST></VAST>"},
				},
			},
		},
	}

	selected, _, err := selector.Select(nil, resp, cfg)
	assert.NoError(t, err)
	require.Len(t, selected, 2)

	// Sequence should be 1-indexed based on position
	assert.Equal(t, 1, selected[0].Sequence)
	assert.Equal(t, 1, selected[0].Meta.SlotInPod)
	assert.Equal(t, 2, selected[1].Sequence)
	assert.Equal(t, 2, selected[1].Meta.SlotInPod)
}

func TestPriceSelector_Select_CanonicalMeta(t *testing.T) {
	selector := NewPriceSelector(1)
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "USD",
		MaxAdsInPod:     5,
	}
	resp := &openrtb2.BidResponse{
		Cur: "EUR",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{
						ID:      "bid1",
						ImpID:   "imp1",
						Price:   2.5,
						AdM:     "<VAST></VAST>",
						DealID:  "deal123",
						ADomain: []string{"advertiser.com", "other.com"},
						Cat:     []string{"IAB1", "IAB2"},
						Dur:     30,
					},
				},
			},
		},
	}

	selected, _, err := selector.Select(nil, resp, cfg)
	assert.NoError(t, err)
	require.Len(t, selected, 1)

	meta := selected[0].Meta
	assert.Equal(t, "bid1", meta.BidID)
	assert.Equal(t, "imp1", meta.ImpID)
	assert.Equal(t, "deal123", meta.DealID)
	assert.Equal(t, "bidder1", meta.Seat)
	assert.Equal(t, 2.5, meta.Price)
	assert.Equal(t, "EUR", meta.Currency) // From response
	assert.Equal(t, "advertiser.com", meta.Adomain)
	assert.Equal(t, []string{"IAB1", "IAB2"}, meta.Cats)
	assert.Equal(t, 30, meta.DurSec)
	assert.Equal(t, 1, meta.SlotInPod)
}

func TestPriceSelector_Select_CurrencyFallback(t *testing.T) {
	selector := NewPriceSelector(1)
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "GBP",
		MaxAdsInPod:     5,
	}
	resp := &openrtb2.BidResponse{
		Cur: "", // Empty currency
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", Price: 1.0, AdM: "<VAST></VAST>"},
				},
			},
		},
	}

	selected, _, err := selector.Select(nil, resp, cfg)
	assert.NoError(t, err)
	require.Len(t, selected, 1)
	assert.Equal(t, "GBP", selected[0].Meta.Currency) // Fallback to config
}

func TestPriceSelector_Select_MultipleSeatBids(t *testing.T) {
	selector := NewPriceSelector(0)
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "USD",
		MaxAdsInPod:     5,
	}
	resp := &openrtb2.BidResponse{
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid1", Price: 1.0, AdM: "<VAST></VAST>"},
				},
			},
			{
				Seat: "bidder2",
				Bid: []openrtb2.Bid{
					{ID: "bid2", Price: 2.0, AdM: "<VAST></VAST>"},
				},
			},
			{
				Seat: "bidder3",
				Bid: []openrtb2.Bid{
					{ID: "bid3", Price: 3.0, AdM: "<VAST></VAST>"},
				},
			},
		},
	}

	selected, _, err := selector.Select(nil, resp, cfg)
	assert.NoError(t, err)
	require.Len(t, selected, 3)

	// Should be sorted by price, with correct seat assignment
	assert.Equal(t, "bid3", selected[0].Meta.BidID)
	assert.Equal(t, "bidder3", selected[0].Seat)
	assert.Equal(t, "bid2", selected[1].Meta.BidID)
	assert.Equal(t, "bidder2", selected[1].Seat)
	assert.Equal(t, "bid1", selected[2].Meta.BidID)
	assert.Equal(t, "bidder1", selected[2].Seat)
}

func TestPriceSelector_Select_ComplexSort(t *testing.T) {
	selector := NewPriceSelector(0)
	cfg := vast.ReceiverConfig{
		DefaultCurrency: "USD",
		MaxAdsInPod:     10,
	}
	resp := &openrtb2.BidResponse{
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "e", Price: 2.0, AdM: "<VAST></VAST>", DealID: ""},      // Same price, no deal
					{ID: "a", Price: 3.0, AdM: "<VAST></VAST>", DealID: "deal1"}, // Highest price with deal
					{ID: "b", Price: 3.0, AdM: "<VAST></VAST>", DealID: ""},      // Highest price, no deal
					{ID: "c", Price: 2.0, AdM: "<VAST></VAST>", DealID: "deal2"}, // Same price with deal
					{ID: "d", Price: 2.0, AdM: "<VAST></VAST>", DealID: "deal3"}, // Same price with deal
					{ID: "f", Price: 1.0, AdM: "<VAST></VAST>", DealID: ""},      // Lowest price
				},
			},
		},
	}

	selected, _, err := selector.Select(nil, resp, cfg)
	assert.NoError(t, err)
	require.Len(t, selected, 6)

	// Expected order:
	// 1. a (price 3.0, deal) - highest price with deal
	// 2. b (price 3.0, no deal) - highest price, no deal
	// 3. c (price 2.0, deal) - same price, deal, ID "c"
	// 4. d (price 2.0, deal) - same price, deal, ID "d"
	// 5. e (price 2.0, no deal) - same price, no deal
	// 6. f (price 1.0) - lowest price
	assert.Equal(t, "a", selected[0].Meta.BidID)
	assert.Equal(t, "b", selected[1].Meta.BidID)
	assert.Equal(t, "c", selected[2].Meta.BidID)
	assert.Equal(t, "d", selected[3].Meta.BidID)
	assert.Equal(t, "e", selected[4].Meta.BidID)
	assert.Equal(t, "f", selected[5].Meta.BidID)
}

func TestNewSingleSelector(t *testing.T) {
	selector := NewSingleSelector()
	require.NotNil(t, selector)

	priceSelector, ok := selector.(*PriceSelector)
	require.True(t, ok)
	assert.Equal(t, 1, priceSelector.maxBids)
}

func TestNewTopNSelector(t *testing.T) {
	selector := NewTopNSelector()
	require.NotNil(t, selector)

	priceSelector, ok := selector.(*PriceSelector)
	require.True(t, ok)
	assert.Equal(t, 0, priceSelector.maxBids)
}
