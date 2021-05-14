package adapters

import (
	"text/template"

	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/usersync"
)

type Syncer struct {
	familyName  string
	syncType    SyncType
	urlTemplate *template.Template
}

func NewSyncer(familyName string, urlTemplate *template.Template, syncType SyncType) *Syncer {
	return &Syncer{
		familyName:  familyName,
		urlTemplate: urlTemplate,
		syncType:    syncType,
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
