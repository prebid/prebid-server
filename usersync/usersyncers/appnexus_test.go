package usersyncers

import (
	"testing"
)

func TestAppNexusSyncer(t *testing.T) {
	an := NewAppnexusSyncer("https://prebid.adnxs.com/pbs/v1")
	syncInfo := an.GetUsersyncInfo("", "")
	assertStringsMatch(t, "//ib.adnxs.com/getuid?https%3A%2F%2Fprebid.adnxs.com%2Fpbs%2Fv1%2Fsetuid%3Fbidder%3Dadnxs%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24UID", syncInfo.URL)
	assertStringsMatch(t, "redirect", syncInfo.Type)
	if syncInfo.SupportCORS != false {
		t.Errorf("should have been false")
	}
	if an.GDPRVendorID() != 32 {
		t.Errorf("Wrong Appnexus GDPR VendorID. Got %d", an.GDPRVendorID())
	}
}
