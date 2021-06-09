package bmtm

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestSyncer(t *testing.T) {
	syncURL := "https://synctest/?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&url=localhost%2Fsetuid%3Fbidder%3Dbmtm%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3DUUID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewBmtmSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "consent-string",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://synctest/?gdpr=1&gdpr_consent=consent-string&url=localhost%2Fsetuid%3Fbidder%3Dbmtm%26gdpr%3D1%26gdpr_consent%3Dconsent-string%26uid%3DUUID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
}
