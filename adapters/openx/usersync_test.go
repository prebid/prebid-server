package openx

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestOpenxSyncer(t *testing.T) {
	syncURL := "https://rtb.openx.net/sync/prebid?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&r=localhost%2Fsetuid%3Fbidder%3Dopenx%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUID%7D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewOpenxSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "https://rtb.openx.net/sync/prebid?gdpr=&gdpr_consent=&r=localhost%2Fsetuid%3Fbidder%3Dopenx%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24%7BUID%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
