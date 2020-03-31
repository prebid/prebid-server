package ninthdecimal

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestninthdecimalSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://rtb.ninthdecimal.com/xp/user-sync?acctid={aid}&&redirect=localhost/setuid?bidder=ninthdecimal&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&uid=$UID"))
	syncer := NewNinthdecimalSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("1", "BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA")
	assert.NoError(t, err)
	assert.Equal(t, "https://rtb.ninthdecimal.com/xp/user-sync?acctid={aid}&&redirect=localhost/setuid?bidder=ninthdecimal&gdpr=1&gdpr_consent=BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA&uid=$UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 61, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
