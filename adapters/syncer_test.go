package adapters

import (
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

func TestSyncers(t *testing.T) {
	syncers := make(map[openrtb_ext.BidderName]usersync.Usersyncer)
	for _, bidderName := range map[string]openrtb_ext.BidderName{} {
		if _, ok := syncers[bidderName]; !ok {
			t.Errorf("no syncer exists for adapter: %s", bidderName)
		}
	}
}

// Bidders may have an ID on the IAB-maintained global vendor list.
// This makes sure that we don't have conflicting IDs among Bidders in our project,
// since that's almost certainly a bug.
func TestVendorIDUniqueness(t *testing.T) {
	syncers := make(map[openrtb_ext.BidderName]usersync.Usersyncer)
	idMap := make(map[uint16]openrtb_ext.BidderName, 0)
	for name, syncer := range syncers {
		id := syncer.GDPRVendorID()
		if id == 0 {
			continue
		}
		if oldName, ok := idMap[id]; ok {
			t.Errorf("gdpr vendor list id: %d used by both %s and %s", id, oldName, name)
		}
		idMap[id] = name
	}
}
