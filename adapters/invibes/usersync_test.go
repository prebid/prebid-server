package invibes

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestInvibesSyncer(t *testing.T) {
	syncURL := "http://localhost:56479/home/getLid?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirectUri=test.com%2Fsetuid%3Fbidder%3Dinvibes%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewInvibesSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "abc",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:56479/home/getLid?gdpr=1&gdpr_consent=abc&us_privacy=&redirectUri=test.com%2Fsetuid%3Fbidder%3Dinvibes%26gdpr%3D1%26gdpr_consent%3Dabc%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)

}
