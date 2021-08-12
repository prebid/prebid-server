package connectad

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestConnectAdSyncer(t *testing.T) {
	syncURL := "https://cdn.connectad.io/connectmyusers.php?gdpr={{.GDPR}}&consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&cb=localhost%2Fsetuid%3Fbidder%3Dconnectad%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewConnectAdSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "fakeconsent",
		},
		CCPA: ccpa.Policy{
			Consent: "fake",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://cdn.connectad.io/connectmyusers.php?gdpr=1&consent=fakeconsent&us_privacy=fake&cb=localhost%2Fsetuid%3Fbidder%3Dconnectad%26gdpr%3D1%26gdpr_consent%3Dfakeconsent%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
