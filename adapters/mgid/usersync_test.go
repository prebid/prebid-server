package mgid

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestMgidSyncer(t *testing.T) {
	syncURL := "https://cm.mgid.com/m?cdsp=363893&adu=https%3A//external.com%2Fsetuid%3Fbidder%3Dmgid%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7Bmuidn%7D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewMgidSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal: "0",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://cm.mgid.com/m?cdsp=363893&adu=https%3A//external.com%2Fsetuid%3Fbidder%3Dmgid%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%7Bmuidn%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
