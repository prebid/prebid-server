package tappx

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestTappxSyncer(t *testing.T) {
	syncURL := "//testing.ssp.tappx.com/cs/usersync.php?gdpr_optin={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&type=iframe&ruid=localhost%2Fsetuid%3Fbidder%3Dtappx%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7B%7BTPPXUID%7D%7D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewTappxSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "A",
		},
		CCPA: ccpa.Policy{
			Consent: "1YNN",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//testing.ssp.tappx.com/cs/usersync.php?gdpr_optin=1&gdpr_consent=A&us_privacy=1YNN&type=iframe&ruid=localhost%2Fsetuid%3Fbidder%3Dtappx%26gdpr%3D1%26gdpr_consent%3DA%26uid%3D%7B%7BTPPXUID%7D%7D", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
