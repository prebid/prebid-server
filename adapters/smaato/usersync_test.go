package smaato

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestSmaatoSyncer(t *testing.T) {
	syncURL := "//ads.smaato.com/AdServer/js/user_sync.html?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&predirect=localhost%2Fsetuid%3Fbidder%3Dsmaato%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewSmaatoSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "A",
			Consent: "B",
		},
		CCPA: ccpa.Policy{
			Value: "C",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//ads.smaato.com/AdServer/js/user_sync.html?gdpr=A&gdpr_consent=B&us_privacy=C&predirect=localhost%2Fsetuid%3Fbidder%3Dsmaato%26gdpr%3DA%26gdpr_consent%3DB%26uid%3D", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 76, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
