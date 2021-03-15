package cpmstar

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestCpmstarSyncer(t *testing.T) {
	syncURL := "https://server.cpmstar.com/usersync.aspx?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirectUri=http%3A%2F%2Flocalhost:8000%2Fsetuid%3Fbidder%3Dcpmstar%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26us_privacy%3D{{.USPrivacy}}%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewCpmstarSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "https://server.cpmstar.com/usersync.aspx?gdpr=&gdpr_consent=&us_privacy=&redirectUri=http%3A%2F%2Flocalhost:8000%2Fsetuid%3Fbidder%3Dcpmstar%26gdpr%3D%26gdpr_consent%3D%26us_privacy%3D%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.False(t, syncInfo.SupportCORS)
}
