package gamma

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestGammaSyncer(t *testing.T) {
	syncURL := "//hb.gammaplatform.com/sync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&redirectUri=http%3A%2F%2Flocalhost%2F%2Fsetuid%3Fbidder%3Dgamma%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewGammaSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal: "0",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//hb.gammaplatform.com/sync?gdpr=0&gdpr_consent=&redirectUri=http%3A%2F%2Flocalhost%2F%2Fsetuid%3Fbidder%3Dgamma%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.False(t, syncInfo.SupportCORS)
}
