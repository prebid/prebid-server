package sovrn

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestSovrnSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("//ap.lijit.com/pixel?redir=external.com%2Fsetuid%3Fbidder%3Dsovrn%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"))
	syncer := NewSovrnSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("0", "")
	assert.NoError(t, err)
	assert.Equal(t, "//ap.lijit.com/pixel?redir=external.com%2Fsetuid%3Fbidder%3Dsovrn%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 13, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
