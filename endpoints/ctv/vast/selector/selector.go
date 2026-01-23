package selector

import (
	"errors"
	"sort"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/exchange/entities"
)

// Strategy defines the bid selection strategy
type Strategy string

const (
	// StrategySingle selects the highest priced bid
	StrategySingle Strategy = "SINGLE"
	// StrategyTopN selects the top N bids for ad pods
	StrategyTopN Strategy = "TOP_N"
)

// Config holds selector configuration
type Config struct {
	Strategy    Strategy
	MaxAdsInPod int
}

// BidWithSeat wraps a bid with its seat information
type BidWithSeat struct {
	Bid  *openrtb2.Bid
	Seat string
}

// SelectResult contains the selected bids
type SelectResult struct {
	Bids     []*BidWithSeat
	Rejected []RejectedBid
}

// RejectedBid contains information about a rejected bid
type RejectedBid struct {
	Bid    *BidWithSeat
	Reason string
}

// Selector selects winning bids from auction response
type Selector interface {
	// Select chooses winning bids based on strategy
	Select(response *openrtb2.BidResponse, config Config) (*SelectResult, error)
}

// DefaultSelector implements the Selector interface
type DefaultSelector struct{}

// NewSelector creates a new DefaultSelector
func NewSelector() Selector {
	return &DefaultSelector{}
}

// Select implements Selector.Select
func (s *DefaultSelector) Select(response *openrtb2.BidResponse, config Config) (*SelectResult, error) {
	if response == nil {
		return nil, errors.New("bid response is nil")
	}

	if config.MaxAdsInPod < 1 {
		config.MaxAdsInPod = 1
	}

	// Collect all bids with seat information
	allBids := s.collectBids(response)

	if len(allBids) == 0 {
		return &SelectResult{
			Bids:     []*BidWithSeat{},
			Rejected: []RejectedBid{},
		}, nil
	}

	// Sort bids by price (highest first)
	s.sortBidsByPrice(allBids)

	var selected []*BidWithSeat
	var rejected []RejectedBid

	switch config.Strategy {
	case StrategySingle:
		// Select single highest priced bid
		selected = append(selected, allBids[0])
		for i := 1; i < len(allBids); i++ {
			rejected = append(rejected, RejectedBid{
				Bid:    allBids[i],
				Reason: "not_highest_price",
			})
		}

	case StrategyTopN:
		// Select top N bids
		maxSelect := config.MaxAdsInPod
		if maxSelect > len(allBids) {
			maxSelect = len(allBids)
		}

		selected = allBids[:maxSelect]
		for i := maxSelect; i < len(allBids); i++ {
			rejected = append(rejected, RejectedBid{
				Bid:    allBids[i],
				Reason: "exceeded_max_ads_in_pod",
			})
		}

	default:
		// Default to single selection
		selected = append(selected, allBids[0])
		for i := 1; i < len(allBids); i++ {
			rejected = append(rejected, RejectedBid{
				Bid:    allBids[i],
				Reason: "not_highest_price",
			})
		}
	}

	return &SelectResult{
		Bids:     selected,
		Rejected: rejected,
	}, nil
}

// collectBids extracts all bids from the response with seat information
func (s *DefaultSelector) collectBids(response *openrtb2.BidResponse) []*BidWithSeat {
	var bids []*BidWithSeat

	for _, seatBid := range response.SeatBid {
		seat := seatBid.Seat
		for i := range seatBid.Bid {
			bids = append(bids, &BidWithSeat{
				Bid:  &seatBid.Bid[i],
				Seat: seat,
			})
		}
	}

	return bids
}

// sortBidsByPrice sorts bids by price in descending order
func (s *DefaultSelector) sortBidsByPrice(bids []*BidWithSeat) {
	sort.Slice(bids, func(i, j int) bool {
		// Sort by price (highest first)
		if bids[i].Bid.Price != bids[j].Bid.Price {
			return bids[i].Bid.Price > bids[j].Bid.Price
		}
		
		// If prices are equal, use bid ID for stable sorting
		return bids[i].Bid.ID < bids[j].Bid.ID
	})
}

// SelectFromPbsResponse is a helper for PBS internal types
func SelectFromPbsResponse(seatBids map[string]*entities.PbsOrtbSeatBid, config Config) (*SelectResult, error) {
	// Convert PBS internal format to OpenRTB format
	response := &openrtb2.BidResponse{
		SeatBid: make([]openrtb2.SeatBid, 0, len(seatBids)),
	}

	for seat, pbsSeatBid := range seatBids {
		if pbsSeatBid == nil || len(pbsSeatBid.Bids) == 0 {
			continue
		}

		seatBid := openrtb2.SeatBid{
			Seat: seat,
			Bid:  make([]openrtb2.Bid, 0, len(pbsSeatBid.Bids)),
		}

		for _, pbsBid := range pbsSeatBid.Bids {
			if pbsBid != nil && pbsBid.Bid != nil {
				seatBid.Bid = append(seatBid.Bid, *pbsBid.Bid)
			}
		}

		if len(seatBid.Bid) > 0 {
			response.SeatBid = append(response.SeatBid, seatBid)
		}
	}

	selector := NewSelector()
	return selector.Select(response, config)
}

// GetSequence returns the sequence number for a bid (for ad pods)
// Uses SlotInPod extension if available, otherwise uses index
func GetSequence(bid *openrtb2.Bid, index int) int {
	// Try to extract SlotInPod from bid extensions if needed
	// For MVP, just use index + 1
	if index < 0 {
		return 1
	}
	return index + 1
}
