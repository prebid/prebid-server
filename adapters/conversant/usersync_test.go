package conversant

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestConversantSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("usersync?rurl=localhost%2Fsetuid%3Fbidder%3Dconversant%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D"))
	syncer := NewConversantSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("0", "")
	assert.NoError(t, err)
	assert.Equal(t, "usersync?rurl=localhost%2Fsetuid%3Fbidder%3Dconversant%26gdpr%3D0%26gdpr_consent%3D%26uid%3D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 24, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
