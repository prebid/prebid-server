package usersyncers

import (
	"testing"
)

func TestConversantSyncer(t *testing.T) {
	syncer := NewConversantSyncer("usersync?rurl=", "localhost")
	info := syncer.GetUsersyncInfo("0", "")

	uri := "usersync?rurl=localhost%2Fsetuid%3Fbidder%3Dconversant%26gdpr%3D0%26gdpr_consent%3D%26uid%3D"
	assertStringsMatch(t, uri, info.URL)
	assertStringsMatch(t, "redirect", info.Type)

	if info.SupportCORS != false {
		t.Fatalf("user sync should not support CORS")
	}
	if syncer.GDPRVendorID() != 24 {
		t.Errorf("Wrong Conversant GDPR VendorID. Got %d", syncer.GDPRVendorID())
	}
}
