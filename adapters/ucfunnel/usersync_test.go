package ucfunnel

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestUcfunnelSyncer(t *testing.T) {
	syncURL := "//sync.aralego.com/idsync?gdpr={{.GDPR}}&redirect=externalURL.com%2Fsetuid%3Fbidder%3Ducfunnel%26uid%3DSspCookieUserId"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewUcfunnelSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal: "0",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//sync.aralego.com/idsync?gdpr=0&redirect=externalURL.com%2Fsetuid%3Fbidder%3Ducfunnel%26uid%3DSspCookieUserId", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
