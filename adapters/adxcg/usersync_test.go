package adxcg

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestAdxcgSyncer(t *testing.T) {
	syncURL := "https://app.adxcg.net/cma/cm-notify?pi=prebidsrvtst&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewAdxcgSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal: "0",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://app.adxcg.net/cma/cm-notify?pi=prebidsrvtst&gdpr=0&gdpr_consent=", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
