package adf

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
)

func TestAdfSyncer(t *testing.T) {
	syncURL := "https://cm.adform.net?return_url=localhost%2Fsetuid%3Fbidder%3Dadf%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewAdfSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "A",
			Consent: "B",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://cm.adform.net?return_url=localhost%2Fsetuid%3Fbidder%3Dadf%26gdpr%3DA%26gdpr_consent%3DB%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
