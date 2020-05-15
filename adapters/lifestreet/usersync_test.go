package lifestreet

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestLifestreetSyncer(t *testing.T) {
	syncURL := "//ads.lfstmedia.com/idsync/137062?synced=1&ttl=1s&rurl=localhost%2Fsetuid%3Fbidder%3Dlifestreet%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%24visitor_cookie%24%24"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewLifestreetSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal: "0",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//ads.lfstmedia.com/idsync/137062?synced=1&ttl=1s&rurl=localhost%2Fsetuid%3Fbidder%3Dlifestreet%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24%24visitor_cookie%24%24", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 67, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
