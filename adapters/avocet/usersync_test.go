package avocet

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestAvocetSyncer(t *testing.T) {
	syncURL := "https://ads.avct.cloud/getuid?&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&url=%2Fsetuid%3Fbidder%3Davocet%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7B%7BUUID%7D%7D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewAvocetSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "https://ads.avct.cloud/getuid?&gdpr=&gdpr_consent=&us_privacy=&url=%2Fsetuid%3Fbidder%3Davocet%26gdpr%3D%26gdpr_consent%3D%26uid%3D%7B%7BUUID%7D%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 63, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
