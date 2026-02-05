// Package bidselect provides bid selection logic for CTV VAST ad pods.
package bidselect

import (
	"github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment"
)

// Selector implements the vast.BidSelector interface.
// It provides factory methods for different selection strategies.
type Selector interface {
	vast.BidSelector
}

// NewSelector creates a BidSelector based on the selection strategy.
// Supported strategies:
//   - "SINGLE": Returns a single best bid (PriceSelector with limit 1)
//   - "TOP_N": Returns up to MaxAdsInPod bids (PriceSelector)
//   - Default: Falls back to TOP_N behavior
func NewSelector(strategy vast.SelectionStrategy) Selector {
	switch strategy {
	case vast.SelectionSingle:
		return NewPriceSelector(1)
	case vast.SelectionTopN:
		return NewPriceSelector(0) // 0 means use cfg.MaxAdsInPod
	default:
		// Default to TOP_N behavior for unknown strategies
		return NewPriceSelector(0)
	}
}

// NewSingleSelector creates a selector that returns only the best bid.
func NewSingleSelector() Selector {
	return NewPriceSelector(1)
}

// NewTopNSelector creates a selector that returns up to MaxAdsInPod bids.
func NewTopNSelector() Selector {
	return NewPriceSelector(0)
}

// Ensure PriceSelector implements Selector interface.
var _ Selector = (*PriceSelector)(nil)
