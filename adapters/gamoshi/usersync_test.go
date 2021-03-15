package gamoshi

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/stretchr/testify/assert"
)

func TestGamoshiSyncer(t *testing.T) {
	syncURL := "https://rtb.gamoshi.io/user_sync_prebid?gdpr={{.GDPR}}&consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&rurl=https%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dgamoshi%26gdpr%3D%7B%7B.GDPR%7D%7D%26gdpr_consent%3D%7B%7B.GDPRConsent%7D%7D%26uid%3D%5Bgusr%5D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewGamoshiSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		CCPA: ccpa.Policy{
			Consent: "anyValue",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://rtb.gamoshi.io/user_sync_prebid?gdpr=&consent=&us_privacy=anyValue&rurl=https%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dgamoshi%26gdpr%3D%7B%7B.GDPR%7D%7D%26gdpr_consent%3D%7B%7B.GDPRConsent%7D%7D%26uid%3D%5Bgusr%5D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
