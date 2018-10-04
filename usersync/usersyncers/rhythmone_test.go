package usersyncers

import (
	"testing"
)

func TestRhythmoneSyncer(t *testing.T) {
	an := NewRhythmoneSyncer("https://sync.1rx.io/usersync2/rmphb?redir=", "localhost")
	syncInfo := an.GetUsersyncInfo("", "")
	url := "https://sync.1rx.io/usersync2/rmphb?redir=localhost%2Fsetuid%3Fbidder%3Drhythmone%26gdpr%3D%7B%7Bgdpr%7D%7D%26gdpr_consent%3D%7B%7Bgdpr_consent%7D%7D%26uid%3D%7BRX_UUID%7D"
	assertStringsMatch(t, url, syncInfo.URL)
	assertStringsMatch(t, "redirect", syncInfo.Type)
	if syncInfo.SupportCORS != false {
		t.Errorf("should have been false")
	}
	if an.GDPRVendorID() != 36 {
		t.Errorf("Wrong Rhythmone GDPR VendorID. Got %d", an.GDPRVendorID())
	}
}
