package lunamedia

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestLunaMediaSyncer(t *testing.T) {
	syncURL := "https://api.lunamedia.io/xp/user-sync?acctid={aid}&&redirect=localhost/setuid?bidder=lunamedia&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&uid=$UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewLunaMediaSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "A",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://api.lunamedia.io/xp/user-sync?acctid={aid}&&redirect=localhost/setuid?bidder=lunamedia&gdpr=1&gdpr_consent=A&uid=$UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
