package gamoshi

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestGamoshiSyncer(t *testing.T) {
	syncURL := "https://rtb.gamoshi.io/pix/1707/scm?gdpr={{.GDPR}}&consent={{.GDPRConsent}}&rurl=localhost/setuid%3Fbidder%3Dgamoshi%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5Bgusr%5D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewGamoshiSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "https://rtb.gamoshi.io/pix/1707/scm?gdpr=&consent=&rurl=localhost/setuid%3Fbidder%3Dgamoshi%26gdpr%3D%26gdpr_consent%3D%26uid%3D%5Bgusr%5D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 644, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
