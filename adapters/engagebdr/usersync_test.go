package engagebdr

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestEngageBDRSyncer(t *testing.T) {
	syncURL := "https://match.bnmla.com/usersync/s2s?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&r=localhost%2Fsetuid%3Fbidder%3Dengagebdr%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewEngageBDRSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA",
		},
		CCPA: ccpa.Policy{
			Consent: "1NYN",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://match.bnmla.com/usersync/s2s?gdpr=1&gdpr_consent=BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA&us_privacy=1NYN&r=localhost%2Fsetuid%3Fbidder%3Dengagebdr%26gdpr%3D1%26gdpr_consent%3DBOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA%26uid%3D", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 62, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
