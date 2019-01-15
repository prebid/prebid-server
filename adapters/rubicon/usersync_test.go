package rubicon

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestRubiconSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://pixel.rubiconproject.com/exchange/sync.php?p=prebid&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}"))
	syncer := NewRubiconSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("0", "")
	assert.NoError(t, err)
	assert.Equal(t, "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid&gdpr=0&gdpr_consent=", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 52, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
	assert.Equal(t, "rubicon", syncer.FamilyName())
}
