package operaads

import (
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestOperaadsSyncer(t *testing.T) {
	syncURL := "https://t-odx.op-mobile.opera.com/pbs/sync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&r=localhost%2Fsetuid%3Fbidder%3Doperaads%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUID%7D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewOperaadsSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "A",
			Consent: "B",
		},
		CCPA: ccpa.Policy{
			Consent: "C",
		}})

	assert.NoError(t, err)
	assert.Equal(t, "https://t-odx.op-mobile.opera.com/pbs/sync?gdpr=A&gdpr_consent=B&r=localhost%2Fsetuid%3Fbidder%3Doperaads%26gdpr%3DA%26gdpr_consent%3DB%26uid%3D%24%7BUID%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
