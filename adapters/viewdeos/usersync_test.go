package viewdeos

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestViewdeosSyncer(t *testing.T) {
	syncURL := "//sync.sync.viewdeos.com/csync?t=p&ep=0&redir=localhost%2Fsetuid%3Fbidder%3Dmediafuse%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7Buid%7D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewViewdeosSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal: "0",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//sync.sync.viewdeos.com/csync?t=p&ep=0&redir=localhost%2Fsetuid%3Fbidder%3Dmediafuse%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%7Buid%7D", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
