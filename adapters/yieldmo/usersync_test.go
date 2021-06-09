package yieldmo

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestYieldmoSyncer(t *testing.T) {
	syncURL := "//ads.yieldmo.com/pbsync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirectUri=http%3A%2F%2Flocalhost%2F%2Fsetuid%3Fbidder%3Dyieldmo%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewYieldmoSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal: "0",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//ads.yieldmo.com/pbsync?gdpr=0&gdpr_consent=&us_privacy=&redirectUri=http%3A%2F%2Flocalhost%2F%2Fsetuid%3Fbidder%3Dyieldmo%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.False(t, syncInfo.SupportCORS)
}
