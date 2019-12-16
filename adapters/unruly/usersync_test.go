package unruly

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestUnrulySyncer(t *testing.T) {
	syncURL := "//unrulymedia.com/pixel?redir=external.com%2Fsetuid%3Fbidder%3Dunruly%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewUnrulySyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal: "0",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "//unrulymedia.com/pixel?redir=external.com%2Fsetuid%3Fbidder%3Dunruly%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 162, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
