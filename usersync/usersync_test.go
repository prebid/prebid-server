package usersync

import (
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/prebid/prebid-server/config"
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

func TestSyncerVendorIDs(t *testing.T) {
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
