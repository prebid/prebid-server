package usersyncers

import (
	"strings"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestNewSyncerMap(t *testing.T) {
	syncConfig := config.Adapter{
		UserSyncURL: "some-sync-url",
	}
	cfg := &config.Configuration{
		Adapters: map[string]config.Adapter{
			string(openrtb_ext.Bidder33Across):         syncConfig,
			string(openrtb_ext.BidderAdform):           syncConfig,
			string(openrtb_ext.BidderAdkernel):         syncConfig,
			string(openrtb_ext.BidderAdkernelAdn):      syncConfig,
			string(openrtb_ext.BidderAdman):            syncConfig,
			string(openrtb_ext.BidderAdmixer):          syncConfig,
			string(openrtb_ext.BidderAdOcean):          syncConfig,
			string(openrtb_ext.BidderAdpone):           syncConfig,
			string(openrtb_ext.BidderAdtarget):         syncConfig,
			string(openrtb_ext.BidderAdtelligent):      syncConfig,
			string(openrtb_ext.BidderAdvangelists):     syncConfig,
			string(openrtb_ext.BidderAJA):              syncConfig,
			string(openrtb_ext.BidderAppnexus):         syncConfig,
			string(openrtb_ext.BidderAvocet):           syncConfig,
			string(openrtb_ext.BidderBeachfront):       syncConfig,
			string(openrtb_ext.BidderBeintoo):          syncConfig,
			string(openrtb_ext.BidderBrightroll):       syncConfig,
			string(openrtb_ext.BidderColossus):         syncConfig,
			string(openrtb_ext.BidderConnectAd):        syncConfig,
			string(openrtb_ext.BidderConsumable):       syncConfig,
			string(openrtb_ext.BidderConversant):       syncConfig,
			string(openrtb_ext.BidderCpmstar):          syncConfig,
			string(openrtb_ext.BidderDatablocks):       syncConfig,
			string(openrtb_ext.BidderDmx):              syncConfig,
			string(openrtb_ext.BidderEmxDigital):       syncConfig,
			string(openrtb_ext.BidderEngageBDR):        syncConfig,
			string(openrtb_ext.BidderEPlanning):        syncConfig,
			string(openrtb_ext.BidderFacebook):         syncConfig,
			string(openrtb_ext.BidderGamma):            syncConfig,
			string(openrtb_ext.BidderGamoshi):          syncConfig,
			string(openrtb_ext.BidderGrid):             syncConfig,
			string(openrtb_ext.BidderGumGum):           syncConfig,
			string(openrtb_ext.BidderImprovedigital):   syncConfig,
			string(openrtb_ext.BidderInvibes):          syncConfig,
			string(openrtb_ext.BidderIx):               syncConfig,
			string(openrtb_ext.BidderKrushmedia):       syncConfig,
			string(openrtb_ext.BidderLifestreet):       syncConfig,
			string(openrtb_ext.BidderLockerDome):       syncConfig,
			string(openrtb_ext.BidderLogicad):          syncConfig,
			string(openrtb_ext.BidderLunaMedia):        syncConfig,
			string(openrtb_ext.BidderMarsmedia):        syncConfig,
			string(openrtb_ext.BidderMgid):             syncConfig,
			string(openrtb_ext.BidderNanoInteractive):  syncConfig,
			string(openrtb_ext.BidderNinthDecimal):     syncConfig,
			string(openrtb_ext.BidderOpenx):            syncConfig,
			string(openrtb_ext.BidderPubmatic):         syncConfig,
			string(openrtb_ext.BidderPulsepoint):       syncConfig,
			string(openrtb_ext.BidderRhythmone):        syncConfig,
			string(openrtb_ext.BidderRTBHouse):         syncConfig,
			string(openrtb_ext.BidderRubicon):          syncConfig,
			string(openrtb_ext.BidderSharethrough):     syncConfig,
			string(openrtb_ext.BidderSomoaudience):     syncConfig,
			string(openrtb_ext.BidderSonobi):           syncConfig,
			string(openrtb_ext.BidderSovrn):            syncConfig,
			string(openrtb_ext.BidderSmartadserver):    syncConfig,
			string(openrtb_ext.BidderSmartRTB):         syncConfig,
			string(openrtb_ext.BidderSmartyAds):        syncConfig,
			string(openrtb_ext.BidderSynacormedia):     syncConfig,
			string(openrtb_ext.BidderTelaria):          syncConfig,
			string(openrtb_ext.BidderTriplelift):       syncConfig,
			string(openrtb_ext.BidderTripleliftNative): syncConfig,
			string(openrtb_ext.BidderUcfunnel):         syncConfig,
			string(openrtb_ext.BidderUnruly):           syncConfig,
			string(openrtb_ext.BidderValueImpression):  syncConfig,
			string(openrtb_ext.BidderYieldlab):         syncConfig,
			string(openrtb_ext.BidderVerizonMedia):     syncConfig,
			string(openrtb_ext.BidderVisx):             syncConfig,
			string(openrtb_ext.BidderVrtcal):           syncConfig,
			string(openrtb_ext.BidderYieldmo):          syncConfig,
			string(openrtb_ext.BidderYieldone):         syncConfig,
			string(openrtb_ext.BidderZeroClickFraud):   syncConfig,
		},
	}

	adaptersWithoutSyncers := map[openrtb_ext.BidderName]bool{
		openrtb_ext.BidderAdgeneration: true,
		openrtb_ext.BidderAdhese:       true,
		openrtb_ext.BidderAdoppler:     true,
		openrtb_ext.BidderApplogy:      true,
		openrtb_ext.BidderInMobi:       true,
		openrtb_ext.BidderKidoz:        true,
		openrtb_ext.BidderKubient:      true,
		openrtb_ext.BidderMobileFuse:   true,
		openrtb_ext.BidderOrbidder:     true,
		openrtb_ext.BidderPubnative:    true,
		openrtb_ext.BidderSmaato:       true,
		openrtb_ext.BidderTappx:        true,
		openrtb_ext.BidderYeahmobi:     true,
		openrtb_ext.BidderAdprime:      true,
		openrtb_ext.BidderBetween:      true,
	}

	for bidder, config := range cfg.Adapters {
		delete(cfg.Adapters, bidder)
		cfg.Adapters[strings.ToLower(string(bidder))] = config
	}

	syncers := NewSyncerMap(cfg)
	for _, bidderName := range openrtb_ext.BidderMap {
		_, adapterWithoutSyncer := adaptersWithoutSyncers[bidderName]
		if _, ok := syncers[bidderName]; !ok && !adapterWithoutSyncer {
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
