package usersyncers

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRhythmoneSyncer(t *testing.T) {
	assert := assert.New(t)
	an := NewRhythmoneSyncer("https://sync.1rx.io/usersync2/rmphb?gdpr={{gdpr}}&gdpr_consent={{gdpr_consent}}&redir=", "localhost")
	syncInfo := an.GetUsersyncInfo("1", "BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA")
	url := "https://sync.1rx.io/usersync2/rmphb?gdpr=1&gdpr_consent=BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA&redir=localhost%2Fsetuid%3Fbidder%3Drhythmone%26gdpr%3D1%26gdpr_consent%3DBOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA%26uid%3D%5BRX_UUID%5D"
	assert.Equal(url, syncInfo.URL)
	assert.Equal("redirect", syncInfo.Type)
	if syncInfo.SupportCORS != false {
		t.Errorf("should have been false")
	}
	if an.GDPRVendorID() != 36 {
		t.Errorf("Wrong Rhythmone GDPR VendorID. Got %d", an.GDPRVendorID())
	}
}
