package unruly

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestUnrulySyncer(t *testing.T) {
	syncURL := "https://usermatch.targeting.unrulymedia.com/pbsync?gdpr={{.GDPR}}&consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&rurl=%2Fsetuid%3Fbidder%3Dunruly%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewUnrulySyncer(syncURLTemplate)
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
	assert.Equal(t, "https://usermatch.targeting.unrulymedia.com/pbsync?gdpr=A&consent=B&us_privacy=C&rurl=%2Fsetuid%3Fbidder%3Dunruly%26gdpr%3DA%26gdpr_consent%3DB%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
