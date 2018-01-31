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
func TestAdapterList(t *testing.T) {
	list := AdapterList()
	for _, bidderName := range openrtb_ext.BidderMap {
		adapterInList(t, bidderName, list)
	}
}

func adapterInList(t *testing.T, a openrtb_ext.BidderName, l []openrtb_ext.BidderName) {
	found := false
	for _, n := range l {
		if a == n {
			found = true
		}
	}
	if !found {
		t.Errorf("Adapter %s not found in the adapter map!", a)
	}
}
