package usersyncers

import (
	"testing"
)

func TestOathSyncer(t *testing.T) {

	oath := NewOathSyncer("http://east-bid.ybp.yahoo.com/sync/appnexuspbs?url=", "localhost")
	syncInfo := oath.GetUsersyncInfo("", "")
	assertStringsMatch(t, "http%3A%2F%2Feast-bid.ybp.yahoo.com%2Fsync%2Fappnexuspbs%3Furl%3Dlocalhost%2Fsetuid%3Fbidder%3Doath%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24%7BUID%7D", syncInfo.URL)
	assertStringsMatch(t, "redirect", syncInfo.Type)

	if syncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}

	if oath.GDPRVendorID() != 25 {
		t.Errorf("Wrong Oath GDPR VendorID. Got %d", oath.GDPRVendorID())
	}
}
