package adapters

import (
	"text/template"

	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/privacy"
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
	familyName   string
	gdprVendorID uint16
	urlTemplate  *template.Template
	syncType     SyncType
}

func NewSyncer(familyName string, vendorID uint16, urlTemplate *template.Template, syncType SyncType) *Syncer {
	return &Syncer{
		familyName:   familyName,
		gdprVendorID: vendorID,
		urlTemplate:  urlTemplate,
		syncType:     syncType,
	}
}

type SyncType string

const (
	SyncTypeRedirect SyncType = "redirect"
	SyncTypeIframe   SyncType = "iframe"
)

func (s *Syncer) GetUsersyncInfo(privacyPolicies privacy.Policies) (*usersync.UsersyncInfo, error) {
	syncURL, err := macros.ResolveMacros(*s.urlTemplate, macros.UserSyncTemplateParams{
		GDPR:        privacyPolicies.GDPR.Signal,
		GDPRConsent: privacyPolicies.GDPR.Consent,
		USPrivacy:   privacyPolicies.CCPA.Consent,
	})
	if err != nil {
		return nil, err
	}

	return &usersync.UsersyncInfo{
		URL:         syncURL,
		Type:        string(s.syncType),
		SupportCORS: false,
	}, err
}

func (s *Syncer) FamilyName() string {
	return s.familyName
}

func (s *Syncer) GDPRVendorID() uint16 {
	return s.gdprVendorID
}
