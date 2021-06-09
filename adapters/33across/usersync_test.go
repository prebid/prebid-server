package ttx

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func Test33AcrossSyncer(t *testing.T) {
	syncURL := "https://ic.tynt.com/r/d?m=xch&rt=html&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&ru=%2Fsetuid%3Fbidder%3D33across%26uid%3D33XUSERID33X&id=zzz000000000002zzz"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := New33AcrossSyncer(syncURLTemplate)
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
	assert.Equal(t, "https://ic.tynt.com/r/d?m=xch&rt=html&gdpr=A&gdpr_consent=B&us_privacy=C&ru=%2Fsetuid%3Fbidder%3D33across%26uid%3D33XUSERID33X&id=zzz000000000002zzz", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.False(t, syncInfo.SupportCORS)
}
