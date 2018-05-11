package usersyncers

import (
	"testing"
)

func TestIndexSyncer(t *testing.T) {
	syncer := NewIndexSyncer("//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=localhost%2Fsetuid%3Fbidder%3DindexExchange%26uid%3D")
	info := syncer.GetUsersyncInfo()
	if info.URL != "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=localhost%2Fsetuid%3Fbidder%3DindexExchange%26uid%3D" {
		t.Fatalf("should have matched")
	}
	if info.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}
	if syncer.GDPRVendorID() != 10 {
		t.Errorf("Wrong Index GDPR VendorID. Got %d", syncer.GDPRVendorID())
	}
}
