package sonobi

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestSonobiSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("//sync.go.sonobi.com/us.gif?loc=external.com%2Fsetuid%3Fbidder%3Dsonobi%26consent_string%3D{{.GDPR}}%26gdpr%3D{{.GDPRConsent}}%26uid%3D%5BUID%5D"))
	syncer := NewSonobiSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("0", "")
	assert.NoError(t, err)
	assert.Equal(t, "//sync.go.sonobi.com/us.gif?loc=external.com%2Fsetuid%3Fbidder%3Dsonobi%26consent_string%3D0%26gdpr%3D%26uid%3D%5BUID%5D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 104, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
