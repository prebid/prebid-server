package gamoshi

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestGamoshiSyncer(t *testing.T) {

	temp := template.Must(template.New("sync-template").Parse("https://rtb.gamoshi.io/pix/1707/scm?gdpr={{.GDPR}}&consent={{.GDPRConsent}}&rurl=localhost/setuid%3Fbidder%3Dgamoshi%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5Bgusr%5D"))
	syncer := NewGamoshiSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "https://rtb.gamoshi.io/pix/1707/scm?gdpr=&consent=&rurl=localhost/setuid%3Fbidder%3Dgamoshi%26gdpr%3D%26gdpr_consent%3D%26uid%3D%5Bgusr%5D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 644, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
