package usersyncers

import (
	"testing"
)

func TestTtxSyncer(t *testing.T) {
	ttx := NewTtxSyncer("http://localhost", "https://ssc-cms.33across.com/ps", "123")
	syncInfo := ttx.GetUsersyncInfo("", "")
	assertStringsMatch(t, "https://ssc-cms.33across.com/ps/?ri=123&ru=http%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dttx%26gdpr%3D%26gdpr_consent%3D%26uid%3D33XUSERID33X", syncInfo.URL)
	assertStringsMatch(t, "iframe", syncInfo.Type)
	if syncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}
	if ttx.GDPRVendorID() != 999 {
		t.Errorf("Wrong OpenX GDPR VendorID. Got %d", ttx.GDPRVendorID())
	}
}
