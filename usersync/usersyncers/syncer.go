package usersyncers

import (
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

// NewSyncerMap returns a map of all the usersyncer objects.
// The same keys should exist in this map as in the exchanges map.
func NewSyncerMap(cfg *config.Configuration) map[openrtb_ext.BidderName]usersync.Usersyncer {
	return map[openrtb_ext.BidderName]usersync.Usersyncer{
		openrtb_ext.BidderAppnexus:    NewAppnexusSyncer(cfg.ExternalURL),
		openrtb_ext.BidderFacebook:    NewFacebookSyncer(cfg.Adapters["facebook"].UserSyncURL),
		openrtb_ext.BidderConversant:  NewConversantSyncer(cfg.Adapters["conversant"].UserSyncURL, cfg.ExternalURL),
		openrtb_ext.BidderIndex:       NewIndexSyncer(cfg.Adapters["indexexchange"].UserSyncURL),
		openrtb_ext.BidderLifestreet:  NewLifestreetSyncer(cfg.ExternalURL),
		openrtb_ext.BidderOpenx:       NewOpenxSyncer(cfg.ExternalURL),
		openrtb_ext.BidderPubmatic:    NewPubmaticSyncer(cfg.ExternalURL),
		openrtb_ext.BidderPulsepoint:  NewPulsepointSyncer(cfg.ExternalURL),
		openrtb_ext.BidderRubicon:     NewRubiconSyncer(cfg.Adapters["rubicon"].UserSyncURL),
		openrtb_ext.BidderAdform:      NewAdformSyncer(cfg.Adapters["adform"].UserSyncURL, cfg.ExternalURL),
		openrtb_ext.BidderSovrn:       NewSovrnSyncer(cfg.ExternalURL, cfg.Adapters["sovrn"].UserSyncURL),
		openrtb_ext.BidderAdtelligent: NewAdtelligentSyncer(cfg.ExternalURL),
		openrtb_ext.BidderEPlanning:   NewEPlanningSyncer(cfg.Adapters["eplanning"].UserSyncURL, cfg.ExternalURL),
	}
}

type syncer struct {
	familyName   string
	gdprVendorID uint16
	syncInfo     *usersync.UsersyncInfo
}

func (s *syncer) GetUsersyncInfo() *usersync.UsersyncInfo {
	return s.syncInfo
}

func (s *syncer) FamilyName() string {
	return s.familyName
}

func (s *syncer) GDPRVendorID() uint16 {
	return s.gdprVendorID
}
