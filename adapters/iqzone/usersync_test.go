package iqzone

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestIQZoneSyncer(t *testing.T) {
	syncURL := "https://nep.advangelists.com/xp/user-sync?acctid={aid}&&redirect=localhost/setuid?bidder=iqzone&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&uid=$UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewIQZoneSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "A",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://nep.advangelists.com/xp/user-sync?acctid={aid}&&redirect=localhost/setuid?bidder=iqzone&gdpr=1&gdpr_consent=A&uid=$UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 0, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
