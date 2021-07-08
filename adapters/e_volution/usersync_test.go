package evolution

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestNewEvolutionSyncer(t *testing.T) {
	syncURL := "https://sync.test.com/pbserver?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&ccpa={{.USPrivacy}}"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)
	syncer := NewEvolutionSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "0",
			Consent: "allGdpr",
		},
		CCPA: ccpa.Policy{
			Consent: "1---",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://sync.test.com/pbserver?gdpr=0&gdpr_consent=allGdpr&ccpa=1---", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
