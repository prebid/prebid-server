package deepintent

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestDeepintentSyncer(t *testing.T) {
	syncURL := "https://match.deepintent.com/usersync/136?id=unk&gdpr={{.GDPR}}&us_privacy={{.USPrivacy}}&url=https%3A%2F%2Flocalhost%3A8888%2Fsetuid%3Fbidder%3Ddeepintent%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5Bio_cid%5D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewDeepintentSyncer(syncURLTemplate)
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
	assert.Equal(t, "https://match.deepintent.com/usersync/136?id=unk&gdpr=A&us_privacy=C&url=https%3A%2F%2Flocalhost%3A8888%2Fsetuid%3Fbidder%3Ddeepintent%26gdpr%3DA%26gdpr_consent%3DB%26uid%3D%5Bio_cid%5D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
