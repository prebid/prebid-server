package rtbhouse

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestRTBHouseSyncer(t *testing.T) {
	syncURLText := "https://creativecdn.com/cm-notify?pi=prebidsrvtst&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURLText),
	)
	syncer := NewRTBHouseSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo("0", "")

	assert.NoError(t, err)
	assert.Equal(t, "https://creativecdn.com/cm-notify?pi=prebidsrvtst&gdpr=0&gdpr_consent=", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, rtbHouseGDPRVendorID, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
