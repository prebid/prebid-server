package usersyncers

import (
	"testing"
)

func TestRubiconSyncer(t *testing.T) {
	url := "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid&gdpr={{gdpr}}&gdpr_consent={{gdpr_consent}}"

	syncer := NewRubiconSyncer(url)
	info := syncer.GetUsersyncInfo("0", "")

	assertStringsMatch(t, "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid&gdpr=0&gdpr_consent=", info.URL)
	assertStringsMatch(t, "redirect", info.Type)
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}
	assertStringsMatch(t, "rubicon", syncer.FamilyName())
	if syncer.GDPRVendorID() != 52 {
		t.Errorf("Wrong Rubicon GDPR VendorID. Got %d", syncer.GDPRVendorID())
	}
}
