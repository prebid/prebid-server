package logicad

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestLogicadSyncer(t *testing.T) {
	syncURL := "https://localhost/cookiesender?r=true&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&ru=localhost%2Fsetuid%3Fbidder%3Dlogicad%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewLogicadSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "A",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://localhost/cookiesender?r=true&gdpr=1&gdpr_consent=A&ru=localhost%2Fsetuid%3Fbidder%3Dlogicad%26gdpr%3D1%26gdpr_consent%3DA%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
