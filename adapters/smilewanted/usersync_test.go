package smilewanted

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestSmileWantedSyncer(t *testing.T) {
	syncURL := "//csync.smilewanted.com/getuid?source=prebid-server&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirect=https%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dsmilewanted%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewSmileWantedSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "COyASAoOyASAoAfAAAENAfCAAAAAAAAAAAAAAAAAAAAA",
		},
		CCPA: ccpa.Policy{
			Consent: "1YNN",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//csync.smilewanted.com/getuid?source=prebid-server&gdpr=1&gdpr_consent=COyASAoOyASAoAfAAAENAfCAAAAAAAAAAAAAAAAAAAAA&us_privacy=1YNN&redirect=https%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dsmilewanted%26gdpr%3D1%26gdpr_consent%3DCOyASAoOyASAoAfAAAENAfCAAAAAAAAAAAAAAAAAAAAA%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
