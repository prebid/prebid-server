package smartyads

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestSmartyAdsSyncer(t *testing.T) {
	syncURL := "https://as.ck-ie.com/prebid.gif?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir=http%3A%2F%2Flocalhost%3A8000%2Fsetuid%3Fbidder%3Dsmartyads%26uid%3D%5BUID%5D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)
	syncer := NewSmartyAdsSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "0",
			Consent: "ANDFJDS",
		},
		CCPA: ccpa.Policy{
			Consent: "1-YY",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://as.ck-ie.com/prebid.gif?gdpr=0&gdpr_consent=ANDFJDS&us_privacy=1-YY&redir=http%3A%2F%2Flocalhost%3A8000%2Fsetuid%3Fbidder%3Dsmartyads%26uid%3D%5BUID%5D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
