package usersyncers

import (
	"testing"
)

func TestAppNexusSyncer(t *testing.T) {
	an := NewAppnexusSyncer("localhost")
	syncInfo := an.GetUsersyncInfo()
	if syncInfo.URL != "//ib.adnxs.com/getuid?localhost%2Fsetuid%3Fbidder%3Dadnxs%26uid%3D%24UID" {
		t.Errorf("should have matched")
	}
	if syncInfo.Type != "redirect" {
		t.Errorf("should be redirect")
	}
	if syncInfo.SupportCORS != false {
		t.Errorf("should have been false")
	}
	if an.GDPRVendorID() != 32 {
		t.Errorf("Wrong Appnexus GDPR VendorID. Got %d", an.GDPRVendorID())
	}
}
