package selector

import (
	"github.com/prebid/prebid-server/v3/modules/ctv/vast/core"
)

// NewSelector creates a new bid selector based on configuration
// Returns a PriceSelector which implements BidSelector interface
func NewSelector(strategy string) core.BidSelector {
	return &PriceSelector{}
}
