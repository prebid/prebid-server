package mediafuse

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestMediafuseSyncer(t *testing.T) {
	syncURL := "//sync.hbmp.mediafuse.com/csync?t=p&ep=0&redir=localhost%2Fsetuid%3Fbidder%3Dmediafuse%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7Buid%7D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewMediafuseSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal: "0",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//sync.hbmp.mediafuse.com/csync?t=p&ep=0&redir=localhost%2Fsetuid%3Fbidder%3Dmediafuse%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%7Buid%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 411, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
