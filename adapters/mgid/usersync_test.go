package mgid

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"text/template"
)

func TestMgidSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://cm.mgid.com/m?cdsp=363893&adu=https%3A//external.com%2Fsetuid%3Fbidder%3Dmgid%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7Bmuidn%7D"))
	syncer := NewMgidSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("0", "")
	assert.NoError(t, err)
	assert.Equal(t, "https://cm.mgid.com/m?cdsp=363893&adu=https%3A//external.com%2Fsetuid%3Fbidder%3Dmgid%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%7Bmuidn%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 358, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
