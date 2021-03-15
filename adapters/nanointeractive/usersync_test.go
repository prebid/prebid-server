package nanointeractive

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestNewNanoInteractiveSyncer(t *testing.T) {
	syncURL := "https://ad.audiencemanager.de/hbs/cookie_sync?gdpr={{.GDPR}}&consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirectUri=http%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dnanointeractive%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	userSync := NewNanoInteractiveSyncer(syncURLTemplate)
	syncInfo, err := userSync.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw",
		},
		CCPA: ccpa.Policy{
			Consent: "1NYN",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://ad.audiencemanager.de/hbs/cookie_sync?gdpr=1&consent=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw&us_privacy=1NYN&redirectUri=http%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dnanointeractive%26gdpr%3D1%26gdpr_consent%3DBONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
