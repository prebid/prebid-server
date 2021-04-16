package adformOpenRTB

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
)

func TestAdformOpenRTBSyncer(t *testing.T) {
	syncURL := "https://cm.adform.net?return_url=localhost%2Fsetuid%3Fbidder%3Dadform%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26us_privacy%3D{{.USPrivacy}}%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewAdformOpenRTBSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "A",
			Consent: "B",
		},
		CCPA: ccpa.Policy{
			Consent: "C",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://cm.adform.net?return_url=localhost%2Fsetuid%3Fbidder%3Dadform%26gdpr%3DA%26gdpr_consent%3DB%26us_privacy%3DC%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
