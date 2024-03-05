package usersync

import (
	"strings"
)

// BidderFilter determines if a bidder has permission to perform a user sync activity.
type BidderFilter interface {
	// Allowed returns true if the filter determines the bidder has permission and false if either
	// the bidder does not have permission or if the filter has an invalid mode.
	Allowed(bidder string) bool
}

// BidderFilterMode represents the inclusion mode of a BidderFilter.
type BidderFilterMode int

const (
	BidderFilterModeInclude BidderFilterMode = iota
	BidderFilterModeExclude
)

// SpecificBidderFilter implements the BidderFilter which applies the same mode for a list of bidders.
type SpecificBidderFilter struct {
	biddersLookup map[string]struct{}
	mode          BidderFilterMode
}

// Allowed returns true if the bidder is specified and the mode is include or if the bidder is not specified
// and the mode is exclude and returns false in the opposite cases or when the mode is invalid.
func (f SpecificBidderFilter) Allowed(bidder string) bool {
	_, exists := f.biddersLookup[bidder]

	switch f.mode {
	case BidderFilterModeInclude:
		return exists
	case BidderFilterModeExclude:
		return !exists
	default:
		return false
	}
}

// NewSpecificBidderFilter returns a new instance of the NewSpecificBidderFilter filter.
func NewSpecificBidderFilter(bidders []string, mode BidderFilterMode) BidderFilter {
	biddersLookup := make(map[string]struct{}, len(bidders))
	for _, bidder := range bidders {
		biddersLookup[strings.ToLower(bidder)] = struct{}{}
	}

	return SpecificBidderFilter{biddersLookup: biddersLookup, mode: mode}
}

// UniformBidderFilter implements the BidderFilter interface which applies the same mode for all bidders.
type UniformBidderFilter struct {
	mode BidderFilterMode
}

// Allowed returns true if the mode is include and false if the mode is either exclude or invalid.
func (f UniformBidderFilter) Allowed(bidder string) bool {
	return f.mode == BidderFilterModeInclude
}

// NewUniformBidderFilter returns a new instance of the UniformBidderFilter filter.
func NewUniformBidderFilter(mode BidderFilterMode) BidderFilter {
	return UniformBidderFilter{mode: mode}
}
