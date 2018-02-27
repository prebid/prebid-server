package exchange

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestNewAdapterMap(t *testing.T) {
	adapterMap := newAdapterMap(nil, &config.Configuration{})
	for _, bidderName := range openrtb_ext.BidderMap {
		if bidder, ok := adapterMap[bidderName]; bidder == nil || !ok {
			t.Errorf("adapterMap missing expected Bidder: %s", string(bidderName))
		}
	}
}
