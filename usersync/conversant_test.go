package usersync

import (
	"strings"
	"testing"
)

func TestConversantSyncer(t *testing.T) {
	syncer := NewConversantSyncer("usersync?rurl=", "localhost")
	info := syncer.GetUsersyncInfo()

	if !strings.HasSuffix(info.URL, "?rurl=localhost%2Fsetuid%3Fbidder%3Dconversant%26uid%3D") {
		t.Fatalf("bad url suffix. Expected %s got %s", "?rurl=localhost%2Fsetuid%3Fbidder%3Dconversant%26uid%3D", info.URL)
	}

	if info.Type != "redirect" {
		t.Fatalf("user sync type should be redirect: %s", info.Type)
	}

	if info.SupportCORS != false {
		t.Fatalf("user sync should not support CORS")
	}
	if syncer.GDPRVendorID() != 24 {
		t.Errorf("Wrong Conversant GDPR VendorID. Got %d", syncer.GDPRVendorID())
	}
}
