package usersyncers

import "testing"

func TestSovrnSyncer(t *testing.T) {
	syncer := NewSovrnSyncer("external.com", "//ap.lijit.com/pixel?")
	info := syncer.GetUsersyncInfo("0", "")
	assertStringsMatch(t, "//ap.lijit.com/pixel?redir=external.com%2Fsetuid%3Fbidder%3Dsovrn%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24UID", info.URL)
	assertStringsMatch(t, "redirect", info.Type)
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}
	if syncer.GDPRVendorID() != 13 {
		t.Errorf("Wrong Sovrn GDPR VendorID. Got %d", syncer.GDPRVendorID())
	}
}
