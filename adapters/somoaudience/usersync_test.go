package somoaudience

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestSomoaudienceSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("//publisher-east.mobileadtrading.com/usersync?ru=localhost%2Fsetuid%3Fbidder%3Dsomoaudience%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUID%7D"))
	syncer := NewSomoaudienceSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "//publisher-east.mobileadtrading.com/usersync?ru=localhost%2Fsetuid%3Fbidder%3Dsomoaudience%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24%7BUID%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 341, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
