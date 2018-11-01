package adapters

import (
	"strings"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

func GDPRAwareSyncerIDs(syncers map[openrtb_ext.BidderName]usersync.Usersyncer) map[openrtb_ext.BidderName]uint16 {
	gdprAwareSyncers := make(map[openrtb_ext.BidderName]uint16, len(syncers))
	for bidderName, syncer := range syncers {
		if syncer.GDPRVendorID() != 0 {
			gdprAwareSyncers[bidderName] = syncer.GDPRVendorID()
		}
	}
	return gdprAwareSyncers
}

type Syncer struct {
	familyName          string
	gdprVendorID        uint16
	syncEndpointBuilder func(gdpr string, consent string) string
	syncType            SyncType
}

func NewSyncer(familyName string, vendorID uint16, endpointBulder func(gdpr string, consent string) string, syncType SyncType) *Syncer {
	return &Syncer{
		familyName:          familyName,
		gdprVendorID:        vendorID,
		syncEndpointBuilder: endpointBulder,
		syncType:            syncType,
	}
}

type SyncType string

const (
	SyncTypeRedirect SyncType = "redirect"
	SyncTypeIframe   SyncType = "iframe"
)

func (s *Syncer) GetUsersyncInfo(gdpr string, consent string) *usersync.UsersyncInfo {
	return &usersync.UsersyncInfo{
		URL:         s.syncEndpointBuilder(gdpr, consent),
		Type:        string(s.syncType),
		SupportCORS: false,
	}
}

func (s *Syncer) FamilyName() string {
	return s.familyName
}

func (s *Syncer) GDPRVendorID() uint16 {
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
func ResolveMacros(template string) func(gdpr string, consent string) string {
	return func(gdpr string, consent string) string {
		replacer := strings.NewReplacer("{{gdpr}}", gdpr, "{{gdpr_consent}}", consent)
		return replacer.Replace(template)
	}
}
