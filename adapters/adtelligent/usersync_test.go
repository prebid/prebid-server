package adtelligent

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestAdtelligentSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("//sync.adtelligent.com/csync?t=p&ep=0&redir=localhost%2Fsetuid%3Fbidder%3Dadtelligent%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7Buid%7D"))
	syncer := NewAdtelligentSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("0", "")
	assert.NoError(t, err)
	assert.Equal(t, "//sync.adtelligent.com/csync?t=p&ep=0&redir=localhost%2Fsetuid%3Fbidder%3Dadtelligent%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%7Buid%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 0, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
