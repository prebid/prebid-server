package emx_digital

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestEMXDigitalSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("http://cs.emxdgt.com/um?ssp=pbs&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&redirect=localhost%2Fsetuid%3Fbidder%3Demx_digital%26uid%3D%24UID"))
	syncer := NewEMXDigitalSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("1", "BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA")
	assert.NoError(t, err)
	assert.Equal(t, "http%3A%2F%2Fcs.emxdgt.com%2Fum%3Fssp%3Dpbs%26gdpr%3D1%26gdpr_consent%3DBOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA%26redirect%3Dlocalhost%2Fsetuid%3Fbidder%3Demx_digital%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 183, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
