package usersync

import (
	"testing"
)

func TestRubiconSyncer(t *testing.T) {
	url := "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid"

	syncer := NewRubiconSyncer(url)
	info := syncer.GetUsersyncInfo()
	if info.URL != url {
		t.Fatalf("should have matched")
	}
	if info.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}

	if syncer.FamilyName() != "rubicon" {
		t.Errorf("FamilyName '%s' != 'rubicon'", syncer.FamilyName())
	}
	if syncer.GDPRVendorID() != 52 {
		t.Errorf("Wrong Rubicon GDPR VendorID. Got %d", syncer.GDPRVendorID())
	}
}
