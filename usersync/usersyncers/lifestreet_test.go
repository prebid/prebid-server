package usersyncers

import (
	"testing"
)

func TestLifestreetSyncer(t *testing.T) {
	url := "//ads.lfstmedia.com/idsync/137062?synced=1&ttl=1s&rurl=localhost%2Fsetuid%3Fbidder%3Dlifestreet%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24%24visitor_cookie%24%24"

	syncer := NewLifestreetSyncer("localhost")
	info := syncer.GetUsersyncInfo("0", "")
	assertStringsMatch(t, url, info.URL)
	assertStringsMatch(t, "redirect", info.Type)
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}
	if syncer.GDPRVendorID() != 67 {
		t.Errorf("Wrong Lifestreet GDPR VendorID. Got %d", syncer.GDPRVendorID())
	}
}
