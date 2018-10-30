package usersyncers

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestNewSyncerMap(t *testing.T) {
	cfg := &config.Configuration{
		Adapters: map[string]config.Adapter{
			string(openrtb_ext.BidderAdform):       {},
			string(openrtb_ext.BidderAdkernelAdn):  {},
			string(openrtb_ext.BidderAdtelligent):  {},
			string(openrtb_ext.BidderAppnexus):     {},
			string(openrtb_ext.BidderBeachfront):   {},
			string(openrtb_ext.BidderFacebook):     {},
			string(openrtb_ext.BidderBrightroll):   {},
			string(openrtb_ext.BidderConversant):   {},
			string(openrtb_ext.BidderEPlanning):    {},
			string(openrtb_ext.BidderIx):           {},
			string(openrtb_ext.BidderLifestreet):   {},
			string(openrtb_ext.BidderOpenx):        {},
			string(openrtb_ext.BidderPubmatic):     {},
			string(openrtb_ext.BidderPulsepoint):   {},
			string(openrtb_ext.BidderRhythmone):    {},
			string(openrtb_ext.BidderRubicon):      {},
			string(openrtb_ext.BidderSomoaudience): {},
			string(openrtb_ext.BidderSovrn):        {},
		},
	}
	m := NewSyncerMap(cfg)
	if len(m) != len(cfg.Adapters) {
		t.Errorf("length mismatch: expected: %d got: %d", len(m), len(cfg.Adapters))
	}
}

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

func assertStringsMatch(t *testing.T, expected string, actual string) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}
