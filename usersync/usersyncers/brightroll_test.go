package usersyncers

import (
	"testing"
)

func TestBrightrollSyncer(t *testing.T) {

	brightroll := NewBrightrollSyncer("http://east-bid.ybp.yahoo.com/sync/appnexuspbs?gdpr={{gdpr}}&euconsent={{gdpr_consent}}&url=", "localhost")
	syncInfo := brightroll.GetUsersyncInfo("", "")
	assertStringsMatch(t, "http://east-bid.ybp.yahoo.com/sync/appnexuspbs?gdpr=&euconsent=&url=localhost%2Fsetuid%3Fbidder%3Dbrightroll%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24%7BUID%7D", syncInfo.URL)
	assertStringsMatch(t, "redirect", syncInfo.Type)

	if syncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}

	if brightroll.GDPRVendorID() != 25 {
		t.Errorf("Wrong Brightroll(Oath) GDPR VendorID. Got %d", brightroll.GDPRVendorID())
	}
}
