package usersyncers

import (
	"testing"
)

func TestRhythmoneSyncer(t *testing.T) {
	an := NewRhythmoneSyncer("https://sync.1rx.io/usersync2/rmphb?redir=", "localhost")
	syncInfo := an.GetUsersyncInfo("1", "BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA")
	url := "https://sync.1rx.io/usersync2/rmphb?redir=localhost%2Fsetuid%3Fbidder%3Drhythmone%26gdpr%3D1%26gdpr_consent%3DBOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA%26uid%3D%5BRX_UUID%5D"
	assertStringsMatch(t, url, syncInfo.URL)
	assertStringsMatch(t, "redirect", syncInfo.Type)
	if syncInfo.SupportCORS != false {
		t.Errorf("should have been false")
	}
	if an.GDPRVendorID() != 36 {
		t.Errorf("Wrong Rhythmone GDPR VendorID. Got %d", an.GDPRVendorID())
	}
}
