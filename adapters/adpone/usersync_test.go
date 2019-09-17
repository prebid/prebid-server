package adpone

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestAdponeSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://usersync.adpone.com/csync?redir=https%3A%2F%2Flocalhost%3A8888%2Fsetuid%3Fbidder%3Dadpone%26gdpr={{.GDPR&gdpr_consent={{.GDPRConsent}}%26uid%3D%7Buid%7D"))
	syncer := NewadponeSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("1", "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw")
	assert.NoError(t, err)
	assert.Equal(t, "https://usersync.adpone.com/csync?redir=https%3A%2F%2Flocalhost%3A8888%2Fsetuid%3Fbidder%3Dadpone%26gdpr=1&gdpr_consent=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw%26uid%3D%7Buid%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, adponeGDPRVendorID, syncer.GDPRVendorID())
}
