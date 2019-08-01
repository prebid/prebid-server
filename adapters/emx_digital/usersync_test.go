package emx_digital

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestEMXDigitalSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://biddr.brealtime.com/check_pbs.html?redir=localhost%2Fsetuid%3Fbidder%3Demx_digital%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7BUID%7D"))
	syncer := NewEMXDigitalSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("1", "BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA")
	assert.NoError(t, err)
	assert.Equal(t, "https://biddr.brealtime.com/check_pbs.html?redir=localhost%2Fsetuid%3Fbidder%3Demx_digital%26gdpr%3D1%26gdpr_consent%3DBOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA%26uid%3D%7BUID%7D", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 183, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
