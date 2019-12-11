package pubmatic

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestPubmaticSyncer(t *testing.T) {
	syncURL := "//ads.pubmatic.com/AdServer/js/user_sync.html?predirect=localhost%2Fsetuid%3Fbidder%3Dpubmatic%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewPubmaticSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//ads.pubmatic.com/AdServer/js/user_sync.html?predirect=localhost%2Fsetuid%3Fbidder%3Dpubmatic%26gdpr%3D1%26gdpr_consent%3DBONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw%26uid%3D", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 76, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
