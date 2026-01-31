package selector

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/ctv/vast"
)

// Selector implements the BidSelector interface
type Selector struct{}

// NewSelector creates a new bid selector
func NewSelector() vast.BidSelector {
	return &Selector{}
}

// Select implements BidSelector.Select
func (s *Selector) Select(req *openrtb2.BidRequest, resp *openrtb2.BidResponse, cfg vast.ReceiverConfig) ([]vast.SelectedBid, []string, error) {
	// TODO: Implement bid selection logic
	// - Extract bids from response
	// - Apply selection strategy (SINGLE, TOP_N)
	// - Sort by price
	// - Apply maxAdsInPod limit
	// - Extract canonical metadata
	// - Assign sequences
	return nil, nil, nil
}
