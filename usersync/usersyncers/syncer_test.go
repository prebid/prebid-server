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
			string(openrtb_ext.BidderIndex):        {},
			string(openrtb_ext.BidderLifestreet):   {},
			string(openrtb_ext.BidderOpenx):        {},
			string(openrtb_ext.BidderPubmatic):     {},
			string(openrtb_ext.BidderPulsepoint):   {},
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
