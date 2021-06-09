package adpone

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestAdponeSyncer(t *testing.T) {
	syncURL := "https://usersync.adpone.com/csync?t=p&ep=0&redir=localhost%2Fsetuid%3Fbidder%3Dadtelligent%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7Buid%7D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewadponeSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "https://usersync.adpone.com/csync?t=p&ep=0&redir=localhost%2Fsetuid%3Fbidder%3Dadtelligent%26gdpr%3D%26gdpr_consent%3D%26uid%3D%7Buid%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
}
