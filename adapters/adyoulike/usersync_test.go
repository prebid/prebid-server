package adyoulike

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
)

func TestAdyoulikeSyncer(t *testing.T) {
	syncURL := "//visitor.omnitagjs.com/visitor/bsync?uid=19340f4f097d16f41f34fc0274981ca4&name=PrebidServer&gdpr_consent_string={{.GDPRConsent}}&gdpr={{.GDPR}}&us_privacy={{.USPrivacy}}"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewAdyoulikeSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw",
		},
		CCPA: ccpa.Policy{
			Consent: "1-YY",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//visitor.omnitagjs.com/visitor/bsync?uid=19340f4f097d16f41f34fc0274981ca4&name=PrebidServer&gdpr_consent_string=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw&gdpr=1&us_privacy=1-YY", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
