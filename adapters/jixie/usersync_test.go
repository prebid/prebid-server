package jixie

import (
	"github.com/prebid/prebid-server/privacy"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestJixieSyncer(t *testing.T) {
	syncURL := "https://tra.jixie.io/usync/prebidserver?pid=&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirect=https%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Djixie%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7B%7Buid%7D%7D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewJixieSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "https://tra.jixie.io/usync/prebidserver?pid=&gdpr=&gdpr_consent=&us_privacy=&redirect=https%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Djixie%26gdpr%3D%26gdpr_consent%3D%26uid%3D%7B%7Buid%7D%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 0, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
