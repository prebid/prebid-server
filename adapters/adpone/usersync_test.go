package adpone

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestAdponeSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://usersync.adpone.com/csync?t=p&ep=0&redir=localhost%2Fsetuid%3Fbidder%3Dadtelligent%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7Buid%7D"))
	syncer := NewadponeSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "https://usersync.adpone.com/csync?t=p&ep=0&redir=localhost%2Fsetuid%3Fbidder%3Dadtelligent%26gdpr%3D%26gdpr_consent%3D%26uid%3D%7Buid%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, adponeGDPRVendorID, syncer.GDPRVendorID())
}
