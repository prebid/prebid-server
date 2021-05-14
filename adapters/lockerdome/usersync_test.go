package lockerdome

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestLockerDomeSyncer(t *testing.T) {
	syncURL := "https://lockerdome.com/usync/prebidserver?pid=&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirect=https%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dlockerdome%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7B%7Buid%7D%7D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewLockerDomeSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "https://lockerdome.com/usync/prebidserver?pid=&gdpr=&gdpr_consent=&us_privacy=&redirect=https%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dlockerdome%26gdpr%3D%26gdpr_consent%3D%26uid%3D%7B%7Buid%7D%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
