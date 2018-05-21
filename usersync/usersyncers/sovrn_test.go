package usersyncers

import "testing"

func TestSovrnSyncer(t *testing.T) {

	syncer := NewSovrnSyncer("external.com", "sovrn-sync.com")
	info := syncer.GetUsersyncInfo("", "")
	assertStringsMatch(t, "redirect", info.Type)
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}
	if syncer.GDPRVendorID() != 13 {
		t.Errorf("Wrong Sovrn GDPR VendorID. Got %d", syncer.GDPRVendorID())
	}
}
