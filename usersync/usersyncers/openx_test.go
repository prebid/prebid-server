package usersyncers

import (
	"testing"
)

func TestOpenxSyncer(t *testing.T) {
	openx := NewOpenxSyncer("localhost")
	syncInfo := openx.GetUsersyncInfo("", "")
	assertStringsMatch(t, "https://rtb.openx.net/sync/prebid?r=localhost%2Fsetuid%3Fbidder%3Dopenx%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24%7BUID%7D", syncInfo.URL)
	assertStringsMatch(t, "redirect", syncInfo.Type)
	if syncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}
	if openx.GDPRVendorID() != 69 {
		t.Errorf("Wrong OpenX GDPR VendorID. Got %d", openx.GDPRVendorID())
	}
}
