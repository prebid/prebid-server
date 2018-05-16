package usersyncers

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestSyncers(t *testing.T) {
	cfg := &config.Configuration{}
	syncers := NewSyncerMap(cfg)
	for _, bidderName := range openrtb_ext.BidderMap {
		if _, ok := syncers[bidderName]; !ok {
			t.Errorf("No syncer exists for adapter: %s", bidderName)
		}
	}
}

// Bidders may have an ID on the IAB-maintained global vendor list.
// This makes sure that we don't have conflicting IDs among Bidders in our project,
// since that's almost certainly a bug.
func TestVendorIDUniqueness(t *testing.T) {
	cfg := &config.Configuration{}
	syncers := NewSyncerMap(cfg)

	idMap := make(map[uint16]openrtb_ext.BidderName, len(syncers))
	for name, syncer := range syncers {
		id := syncer.GDPRVendorID()
		if id == 0 {
			continue
		}

		if oldName, ok := idMap[id]; ok {
			t.Errorf("GDPR VendorList ID %d used by both %s and %s. These must be unique.", id, oldName, name)
		}
		idMap[id] = name
	}
}
