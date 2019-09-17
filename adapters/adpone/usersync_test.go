package adpone

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestadponeSyncer(t *testing.T) {
	syncURLText := "https://creativecdn.com/cm-notify?pi=prebidsrvtst&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURLText),
	)
	syncer := NewadponeSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo("0", "")

	assert.NoError(t, err)
	assert.Equal(t, "https://creativecdn.com/cm-notify?pi=prebidsrvtst&gdpr=0&gdpr_consent=", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, adponeGDPRVendorID, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
