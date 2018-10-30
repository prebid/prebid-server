package usersyncers

import (
	"testing"
)

func TestIxSyncer(t *testing.T) {
	syncer := NewIxSyncer("//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=localhost%2Fsetuid%3Fbidder%3Dix%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D")
	info := syncer.GetUsersyncInfo("", "")
	assertStringsMatch(t, "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=localhost%2Fsetuid%3Fbidder%3Dix%26gdpr%3D%26gdpr_consent%3D%26uid%3D", info.URL)
	assertStringsMatch(t, "redirect", info.Type)
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}
	if syncer.GDPRVendorID() != 10 {
		t.Errorf("Wrong IX GDPR VendorID. Got %d", syncer.GDPRVendorID())
	}
}
