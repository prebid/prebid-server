package marsmedia

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestMarsmediaSyncer(t *testing.T) {
	syncURL := "https://dmp.rtbsrv.com/dmp/profiles/cm?p_id=179&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirect=localhost:8000%2Fsetuid%3Fbidder%3Dmarsmedia%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUUID%7D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewMarsmediaSyncer(syncURLTemplate)
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
	assert.Equal(t, "https://dmp.rtbsrv.com/dmp/profiles/cm?p_id=179&gdpr=A&gdpr_consent=B&us_privacy=C&redirect=localhost:8000%2Fsetuid%3Fbidder%3Dmarsmedia%26gdpr%3DA%26gdpr_consent%3DB%26uid%3D%24%7BUUID%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)

}
