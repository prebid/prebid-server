package adtarget

import (
	"fmt"
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy/ccpa"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestAdtargetSyncer(t *testing.T) {
	syncURL := "//sync.console.adtarget.com.tr/csync?t=p&ep=0&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir=localhost%2Fsetuid%3Fbidder%3Dadtarget%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7Buid%7D"
	fmt.Println("adtarget sync")
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewAdtargetSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "0",
			Consent: "123",
		},
		CCPA: ccpa.Policy{
			Consent: "1-YY",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//sync.console.adtarget.com.tr/csync?t=p&ep=0&gdpr=0&gdpr_consent=123&us_privacy=1-YY&redir=localhost%2Fsetuid%3Fbidder%3Dadtarget%26gdpr%3D0%26gdpr_consent%3D123%26uid%3D%7Buid%7D", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
