package adyoulike

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
)

func TestAdyoulikeSyncer(t *testing.T) {
	syncURL := "//visitor.omnitagjs.com/broker/bsync?gdpr_consent_string={{.GDPRConsent}}&gdpr={{.GDPR}}"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewAdyoulikeSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//visitor.omnitagjs.com/broker/bsync?gdpr_consent_string=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw&gdpr=1", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 0, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
