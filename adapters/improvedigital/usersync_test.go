package improvedigital

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestImprovedigitalSyncer(t *testing.T) {
	syncURL := "https://ad.360yield.com/server_match?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&r=%2Fsetuid%3Fbidder%3Dimprovedigital%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7BPUB_USER_ID%7D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewImprovedigitalSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "A",
			Consent: "B",
		},
		CCPA: ccpa.Policy{
			Consent: "C",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://ad.360yield.com/server_match?gdpr=A&gdpr_consent=B&us_privacy=C&r=%2Fsetuid%3Fbidder%3Dimprovedigital%26gdpr%3DA%26gdpr_consent%3DB%26uid%3D%7BPUB_USER_ID%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
