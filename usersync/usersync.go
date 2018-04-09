package usersync

import (
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type Usersyncer interface {
	// GetUsersyncInfo returns basic info the browser needs in order to run a user sync.
	// The returned UsersyncInfo object must not be mutated by callers.
	//
	// For more information about user syncs, see http://clearcode.cc/2015/12/cookie-syncing/
	GetUsersyncInfo() *UsersyncInfo
	// FamilyName identifies the space of cookies for this usersyncer.
	// For example, if this Usersyncer syncs with adnxs.com, then this
	// should return "adnxs".
	FamilyName() string
}

type UsersyncInfo struct {
	URL         string `json:"url,omitempty"`
	Type        string `json:"type,omitempty"`
	SupportCORS bool   `json:"supportCORS,omitempty"`
}

type CookieSyncBidders struct {
	BidderCode   string        `json:"bidder"`
	NoCookie     bool          `json:"no_cookie,omitempty"`
	UsersyncInfo *UsersyncInfo `json:"usersync,omitempty"`
}

// NewSyncerMap returns a map of all the usersyncer objects.
// The same keys should exist in this map as in the exchanges map.
func NewSyncerMap(cfg *config.Configuration) map[openrtb_ext.BidderName]Usersyncer {
	return map[openrtb_ext.BidderName]Usersyncer{
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
	}
}

type syncer struct {
	familyName string
	syncInfo   *UsersyncInfo
}

func (s *syncer) GetUsersyncInfo() *UsersyncInfo {
	return s.syncInfo
}

func (s *syncer) FamilyName() string {
	return s.familyName
}
