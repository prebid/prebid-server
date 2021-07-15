package usersync

// BidderFilterMode represents the inclusion mode of a BidderFilter.
type BidderFilterMode int

const (
	BidderFilterModeInclude BidderFilterMode = iota
	BidderFilterModeExclude
)

// BidderFilter determines if a bidder has permission to perform a user sync activity.
type BidderFilter struct {
	biddersAll    bool
	biddersLookup map[string]struct{}
	mode          BidderFilterMode
}

// Allowed returns true if the filter determines the bidder has permission and false if the bidder
// does not have permission or if the BidderFilter is set to an unsupported BidderFilterMode.
func (t BidderFilter) Allowed(bidder string) bool {
	switch t.mode {
	case BidderFilterModeInclude:
		return t.bidderIncluded(bidder)
	case BidderFilterModeExclude:
		return !t.bidderIncluded(bidder)
	default:
		return false
	}
}

func (t BidderFilter) bidderIncluded(bidder string) bool {
	if t.biddersAll {
		return true
	}

	_, exists := t.biddersLookup[bidder]
	return exists
}

// NewBidderFilter returns a new BidderFilter which applies the same mode for a list of specific bidders.
func NewBidderFilter(bidders []string, mode BidderFilterMode) BidderFilter {
	biddersLookup := make(map[string]struct{}, len(bidders))
	for _, bidder := range bidders {
		biddersLookup[bidder] = struct{}{}
	}

	return BidderFilter{biddersLookup: biddersLookup, mode: mode}
}

// NewBidderFilterForAll returns a new BidderFilter which applies the same mode for all bidders.
func NewBidderFilterForAll(mode BidderFilterMode) BidderFilter {
	return BidderFilter{biddersAll: true, mode: mode}
}
