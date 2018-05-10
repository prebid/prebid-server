package usersync

import "testing"

func TestSovrnSyncer(t *testing.T) {

	syncer := NewSovrnSyncer("external.com", "sovrn-sync.com")
	info := syncer.GetUsersyncInfo()
	if info.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}
	if syncer.GDPRVendorID() != 13 {
		t.Errorf("Wrong Sovrn GDPR VendorID. Got %d", syncer.GDPRVendorID())
	}
}
