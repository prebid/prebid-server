package emx_digital

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestEMXDigitalSyncer(t *testing.T) {
	syncURL := "https://cs.emxdgt.com/um?ssp=pbs&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirect=localhost%2Fsetuid%3Fbidder%3Demx_digital%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewEMXDigitalSyncer(syncURLTemplate)
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
	assert.Equal(t, "https://cs.emxdgt.com/um?ssp=pbs&gdpr=1&gdpr_consent=BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA&us_privacy=1NYN&redirect=localhost%2Fsetuid%3Fbidder%3Demx_digital%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
