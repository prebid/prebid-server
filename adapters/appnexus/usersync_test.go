package appnexus

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestAppNexusSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("//ib.adnxs.com/getuid?https%3A%2F%2Fprebid.adnxs.com%2Fpbs%2Fv1%2Fsetuid%3Fbidder%3Dadnxs%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"))
	syncer := NewAppnexusSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "//ib.adnxs.com/getuid?https%3A%2F%2Fprebid.adnxs.com%2Fpbs%2Fv1%2Fsetuid%3Fbidder%3Dadnxs%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 32, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
