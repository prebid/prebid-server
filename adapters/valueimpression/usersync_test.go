package valueimpression

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestValueImpressionSyncer(t *testing.T) {
	syncURL := "https://rtb.valueimpression.com/usersync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&redirectUri=http%3A%2F%2Flocalhost:8000%2Fsetuid%3Fbidder%3Dvalueimpression%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewValueImpressionSyncer(syncURLTemplate)
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
	assert.Equal(t, "https://rtb.valueimpression.com/usersync?gdpr=1&gdpr_consent=BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA&redirectUri=http%3A%2F%2Flocalhost:8000%2Fsetuid%3Fbidder%3Dvalueimpression%26gdpr%3D1%26gdpr_consent%3DBOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.False(t, syncInfo.SupportCORS)
}
