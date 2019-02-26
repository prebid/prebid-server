package gamoshi

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestGamoshiSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("//rtb.gamoshi.io/getuid?https%3A%2F%2Frtb.gamoshi.io/%2Fpbs%2Fv1%2Fsetuid%3Fbidder%3Dadnxs%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"))
	syncer := NewGamoshiSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "//rtb.gamoshi.io/getuid?https%3A%2F%2Frtb.gamoshi.io%2Fpbs%2Fv1%2Fsetuid%3Fbidder%3Dadnxs%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 32, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
