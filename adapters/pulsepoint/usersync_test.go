package pulsepoint

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestPulsepointSyncer(t *testing.T) {
	syncURL := "//bh.contextweb.com/rtset?pid=561205&ev=1&rurl=http%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dpulsepoint%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%25%25VGUID%25%25"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewPulsepointSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "//bh.contextweb.com/rtset?pid=561205&ev=1&rurl=http%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dpulsepoint%26gdpr%3D%26gdpr_consent%3D%26uid%3D%25%25VGUID%25%25", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
