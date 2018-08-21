package usersyncers

import "testing"

func TestSonobiSyncer(t *testing.T) {
	syncer := NewSonobiSyncer("external.com")
	info := syncer.GetUsersyncInfo("0", "")
	assertStringsMatch(t, "//sync.go.sonobi.com/us.gif?loc=external.com%2Fsetuid%3Fbidder%3Dsonobi%26consent_string%3D0%26gdpr%3D%26uid%3D%24UID", info.URL)
	assertStringsMatch(t, "redirect", info.Type)
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}
	if syncer.GDPRVendorID() != 104 {
		t.Errorf("Wrong Sonobi GDPR VendorID. Got %d", syncer.GDPRVendorID())
	}
}
