package between

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestNewBetweenSyncerSyncer(t *testing.T) {
	syncURL := "https://ads.betweendigital.com/match?bidder_id=pbs&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&callback_url=localhost:8080%2Fsetuid%3Fbidder%3Dbetween%26gdpr%3D0%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUSER_ID%7D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewBetweenSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "https://ads.betweendigital.com/match?bidder_id=pbs&gdpr=&gdpr_consent=&us_privacy=&callback_url=localhost:8080%2Fsetuid%3Fbidder%3Dbetween%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24%7BUSER_ID%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
