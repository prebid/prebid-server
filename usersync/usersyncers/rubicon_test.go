package usersyncers

import (
	"testing"
)

func TestRubiconSyncer(t *testing.T) {
	url := "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid"

	syncer := NewRubiconSyncer(url)
	info := syncer.GetUsersyncInfo("", "")

	assertStringsMatch(t, url, info.URL)
	assertStringsMatch(t, "redirect", info.Type)
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}
	assertStringsMatch(t, "rubicon", syncer.FamilyName())
	if syncer.GDPRVendorID() != 52 {
		t.Errorf("Wrong Rubicon GDPR VendorID. Got %d", syncer.GDPRVendorID())
	}
}
