package rhythmone

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestRhythmoneSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://sync.1rx.io/usersync2/rmphb?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&redir=localhost%2Fsetuid%3Fbidder%3Drhythmone%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5BRX_UUID%5D"))
	syncer := NewRhythmoneSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("1", "BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA")
	assert.NoError(t, err)
	assert.Equal(t, "https://sync.1rx.io/usersync2/rmphb?gdpr=1&gdpr_consent=BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA&redir=localhost%2Fsetuid%3Fbidder%3Drhythmone%26gdpr%3D1%26gdpr_consent%3DBOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA%26uid%3D%5BRX_UUID%5D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 36, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
