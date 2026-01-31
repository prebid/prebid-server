package vast

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/prebid/openrtb/v20/openrtb2"
	
)

// PriceSelector implements BidSelector by sorting bids by price
type PriceSelector struct{}

// bidWithSeat is an internal structure for tracking bids with their seat
type bidWithSeat struct {
	bid  *openrtb2.Bid
	seat string
}

// Select implements BidSelector.Select
// Collects bids, filters, sorts by price (desc), dealid (desc), id (asc), and applies strategy
func (s *PriceSelector) Select(req *openrtb2.BidRequest, resp *openrtb2.BidResponse, cfg ReceiverConfig) ([]SelectedBid, []string, error) {
	if resp == nil {
		return nil, []string{"response is nil"}, nil
	}

	warnings := []string{}

	// Collect all bids with their seats
	allBids := s.collectBids(resp)
	if len(allBids) == 0 {
		return nil, []string{"no bids in response"}, nil
	}

	// Filter bids
	validBids := s.filterBids(allBids, cfg, &warnings)
	if len(validBids) == 0 {
		return nil, append(warnings, "no valid bids after filtering"), nil
	}

	// Sort bids by price (desc), dealid exists (desc), id (asc)
	s.sortBids(validBids)

	// Get currency
	currency := s.getCurrency(resp, cfg)

	// Apply selection strategy
	selectedBids := s.applyStrategy(validBids, cfg, currency, &warnings)

	return selectedBids, warnings, nil
}

// collectBids extracts all bids from seatbid array
func (s *PriceSelector) collectBids(resp *openrtb2.BidResponse) []bidWithSeat {
	var result []bidWithSeat

	for _, seatBid := range resp.SeatBid {
		seat := seatBid.Seat
		for i := range seatBid.Bid {
			result = append(result, bidWithSeat{
				bid:  &seatBid.Bid[i],
				seat: seat,
			})
		}
	}

	return result
}

// filterBids removes invalid bids based on configuration
func (s *PriceSelector) filterBids(bids []bidWithSeat, cfg ReceiverConfig, warnings *[]string) []bidWithSeat {
	var valid []bidWithSeat

	for _, b := range bids {
		// Check price > 0
		if b.bid.Price <= 0 {
			*warnings = append(*warnings, fmt.Sprintf("bid %s filtered: price <= 0", b.bid.ID))
			continue
		}

		// Check adm non-empty (unless AllowSkeletonVast is true)
		if !cfg.AllowSkeletonVast && b.bid.AdM == "" {
			*warnings = append(*warnings, fmt.Sprintf("bid %s filtered: empty adm", b.bid.ID))
			continue
		}

		valid = append(valid, b)
	}

	return valid
}

// sortBids sorts by: price desc, dealid exists desc, id asc
func (s *PriceSelector) sortBids(bids []bidWithSeat) {
	sort.Slice(bids, func(i, j int) bool {
		bidI := bids[i].bid
		bidJ := bids[j].bid

		// Primary: price descending
		if bidI.Price != bidJ.Price {
			return bidI.Price > bidJ.Price
		}

		// Secondary: dealid exists descending (deals first)
		hasDealI := bidI.DealID != ""
		hasDealJ := bidJ.DealID != ""
		if hasDealI != hasDealJ {
			return hasDealI // true comes before false
		}

		// Tertiary: bid ID ascending (for stability)
		return bidI.ID < bidJ.ID
	})
}

// getCurrency returns the currency from response or config default
func (s *PriceSelector) getCurrency(resp *openrtb2.BidResponse, cfg ReceiverConfig) string {
	if resp.Cur != "" {
		return resp.Cur
	}
	return cfg.DefaultCurrency
}

// applyStrategy applies SINGLE or TOP_N strategy
func (s *PriceSelector) applyStrategy(bids []bidWithSeat, cfg ReceiverConfig, currency string, warnings *[]string) []SelectedBid {
	if cfg.SelectionStrategy == "SINGLE" {
		return s.selectSingle(bids, currency)
	}
	// Default to TOP_N
	return s.selectTopN(bids, cfg, currency, warnings)
}

// selectSingle returns only the first (highest price) bid
func (s *PriceSelector) selectSingle(bids []bidWithSeat, currency string) []SelectedBid {
	if len(bids) == 0 {
		return nil
	}

	b := bids[0]
	return []SelectedBid{
		{
			Bid:      *b.bid,
			Seat:     b.seat,
			Sequence: 1,
			Meta:     s.extractMetadata(b.bid, b.seat, currency, 1),
		},
	}
}

// selectTopN returns up to MaxAdsInPod bids with sequence assignment
func (s *PriceSelector) selectTopN(bids []bidWithSeat, cfg ReceiverConfig, currency string, warnings *[]string) []SelectedBid {
	maxAds := cfg.MaxAdsInPod
	if maxAds <= 0 {
		maxAds = 10 // Default
	}

	count := len(bids)
	if count > maxAds {
		count = maxAds
		*warnings = append(*warnings, fmt.Sprintf("limited to %d ads (from %d bids)", maxAds, len(bids)))
	}

	result := make([]SelectedBid, count)
	for i := 0; i < count; i++ {
		b := bids[i]
		
		// Determine sequence
		sequence := s.getSequence(b.bid, i)
		
		result[i] = SelectedBid{
			Bid:      *b.bid,
			Seat:     b.seat,
			Sequence: sequence,
			Meta:     s.extractMetadata(b.bid, b.seat, currency, sequence),
		}
	}

	return result
}

// getSequence determines the sequence number for a bid
// Uses bid.Ext.Prebid.Video.SlotInPod if present, otherwise index+1
func (s *PriceSelector) getSequence(bid *openrtb2.Bid, index int) int {
	// Try to get slotInPod from bid extensions
	// For MVP, we'll check bid.Ext for a simple numeric value
	// In production, this would parse bid.ext.prebid.video.slotinpod
	
	// Default to index+1
	return index + 1
}

// extractMetadata builds CanonicalMeta from a bid
func (s *PriceSelector) extractMetadata(bid *openrtb2.Bid, seat string, currency string, slotInPod int) CanonicalMeta {
	meta := CanonicalMeta{
		BidID:     bid.ID,
		ImpID:     bid.ImpID,
		DealID:    bid.DealID,
		Seat:      seat,
		Price:     bid.Price,
		Currency:  currency,
		Adomain:   "",
		Cats:      bid.Cat,
		DurSec:    0,
		SlotInPod: slotInPod,
	}

	// Extract advertiser domain
	if len(bid.ADomain) > 0 {
		meta.Adomain = bid.ADomain[0]
	}

	// Extract duration from bid
	// For video bids, duration might be in various places
	// Try DUR field first (VAST 4.0+)
	if bid.Dur > 0 {
		meta.DurSec = int(bid.Dur)
	}

	// Duration is extracted from Dur field only
	// W field is width and should not be used for duration

	return meta
}

// Helper to parse int from string safely
func parseInt(s string) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}
// NewSelector creates a BidSelector based on the selection strategy
// Currently only supports price-based selection
func NewSelector(strategy string) BidSelector {
// For now, we always return PriceSelector
// In the future, we might support different strategies
return &PriceSelector{}
}
