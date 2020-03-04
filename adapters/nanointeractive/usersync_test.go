package nanointeractive

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestNewNanoInteractiveSyncer(t *testing.T) {
	syncURL := "//ad.audiencemanager.de/hbs/cookieSync/{{.GDPR}}"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	userSync := NewNanoInteractiveSyncer(syncURLTemplate)
	syncInfo, err := userSync.GetUsersyncInfo(
		privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "//ad.audiencemanager.de/hbs/cookieSync/", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 72, userSync.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
