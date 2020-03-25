package admixer

import (
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
	"testing"
	"text/template"
)

func TestAdmixerSyncer(t *testing.T) {
	syncURL := "https://inv-nets.admixer.net/adxcm.aspx?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir=1&rurl=localhost%2Fsetuid%3Fbidder%3Dadmixer%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%24visitor_cookie%24%24"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewAdmixerSyncer(syncURLTemplate)
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
	assert.Equal(t, "https://inv-nets.admixer.net/adxcm.aspx?gdpr=A&gdpr_consent=B&us_privacy=C&redir=1&rurl=localhost%2Fsetuid%3Fbidder%3Dadmixer%26gdpr%3DA%26gdpr_consent%3DB%26uid%3D%24%24visitor_cookie%24%24", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 511, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
