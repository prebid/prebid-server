package usersyncers

import (
	"strings"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

// NewSyncerMap returns a map of all the usersyncer objects.
// The same keys should exist in this map as in the exchanges map.
func NewSyncerMap(cfg *config.Configuration) map[openrtb_ext.BidderName]usersync.Usersyncer {
	return map[openrtb_ext.BidderName]usersync.Usersyncer{
		openrtb_ext.BidderAdform:       NewAdformSyncer(cfg.Adapters[string(openrtb_ext.BidderAdform)].UserSyncURL, cfg.ExternalURL),
		openrtb_ext.BidderAdkernelAdn:  NewAdkernelAdnSyncer(cfg.ExternalURL, cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAdkernelAdn))].UserSyncURL),
		openrtb_ext.BidderAdtelligent:  NewAdtelligentSyncer(cfg.ExternalURL),
		openrtb_ext.BidderAppnexus:     NewAppnexusSyncer(cfg.ExternalURL),
		openrtb_ext.BidderBeachfront:   NewBeachfrontSyncer(cfg.Adapters[string(openrtb_ext.BidderBeachfront)].UserSyncURL, cfg.Adapters[string(openrtb_ext.BidderBeachfront)].PlatformID),
		openrtb_ext.BidderFacebook:     NewFacebookSyncer(cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderFacebook))].UserSyncURL),
		openrtb_ext.BidderBrightroll:   NewBrightrollSyncer(cfg.Adapters[string(openrtb_ext.BidderBrightroll)].UserSyncURL, cfg.ExternalURL),
		openrtb_ext.BidderConversant:   NewConversantSyncer(cfg.Adapters[string(openrtb_ext.BidderConversant)].UserSyncURL, cfg.ExternalURL),
		openrtb_ext.BidderEPlanning:    NewEPlanningSyncer(cfg.Adapters[string(openrtb_ext.BidderEPlanning)].UserSyncURL, cfg.ExternalURL),
		openrtb_ext.BidderIx:           NewIxSyncer(cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderIx))].UserSyncURL),
		openrtb_ext.BidderLifestreet:   NewLifestreetSyncer(cfg.ExternalURL),
		openrtb_ext.BidderOpenx:        NewOpenxSyncer(cfg.ExternalURL),
		openrtb_ext.BidderPubmatic:     NewPubmaticSyncer(cfg.ExternalURL),
		openrtb_ext.BidderPulsepoint:   NewPulsepointSyncer(cfg.ExternalURL),
		openrtb_ext.BidderRhythmone:    NewRhythmoneSyncer(cfg.Adapters[string(openrtb_ext.BidderRhythmone)].UserSyncURL, cfg.ExternalURL),
		openrtb_ext.BidderRubicon:      NewRubiconSyncer(cfg.Adapters[string(openrtb_ext.BidderRubicon)].UserSyncURL),
		openrtb_ext.BidderSomoaudience: NewSomoaudienceSyncer(cfg.ExternalURL),
		openrtb_ext.BidderSovrn:        NewSovrnSyncer(cfg.ExternalURL, cfg.Adapters[string(openrtb_ext.BidderSovrn)].UserSyncURL),
	}
}

func GDPRAwareSyncerIDs(syncers map[openrtb_ext.BidderName]usersync.Usersyncer) map[openrtb_ext.BidderName]uint16 {
	gdprAwareSyncers := make(map[openrtb_ext.BidderName]uint16, len(syncers))
	for bidderName, syncer := range syncers {
		if syncer.GDPRVendorID() != 0 {
			gdprAwareSyncers[bidderName] = syncer.GDPRVendorID()
		}
	}
	return gdprAwareSyncers
}

type syncer struct {
	familyName          string
	gdprVendorID        uint16
	syncEndpointBuilder func(gdpr string, consent string) string
	syncType            SyncType
}

type SyncType string

const (
	SyncTypeRedirect SyncType = "redirect"
	SyncTypeIframe   SyncType = "iframe"
)

func (s *syncer) GetUsersyncInfo(gdpr string, consent string) *usersync.UsersyncInfo {
	return &usersync.UsersyncInfo{
		URL:         s.syncEndpointBuilder(gdpr, consent),
		Type:        string(s.syncType),
		SupportCORS: false,
	}
}

func (s *syncer) FamilyName() string {
	return s.familyName
}

func (s *syncer) GDPRVendorID() uint16 {
	return s.gdprVendorID
}

// This function replaces macros in a sync endpoint template. It will replace:
//
//   {{gdpr}} -- with the "gdpr" string (should be either "0", "1", or "")
//   {{gdpr_consent}} -- with the Raw base64 URL-encoded GDPR Vendor Consent string.
//
// For example, the template:
//   //some-domain.com/getuid?gdpr={{gdpr}}&gdpr_consent={{gdpr_consent}}&callback=prebid-server-domain.com%2Fsetuid%3Fbidder%3Dadnxs%26gdpr={{gdpr}}%26gdpr_consent={{gdpr_consent}}%26uid%3D%24UID
//
// would evaluate to:
//   //some-domain.com/getuid?gdpr=&gdpr_consent=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw&callback=prebid-server-domain.com%2Fsetuid%3Fbidder%3Dadnxs%26gdpr=%26gdpr_consent=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw%26uid%3D%24UID
//
// if the "gdpr" arg was empty, and the consent arg was "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw"
func resolveMacros(template string) func(gdpr string, consent string) string {
	return func(gdpr string, consent string) string {
		replacer := strings.NewReplacer("{{gdpr}}", gdpr, "{{gdpr_consent}}", consent)
		return replacer.Replace(template)
	}
}
