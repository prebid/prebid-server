package usersync

import (
	"testing"
)

func TestOpenxSyncer(t *testing.T) {
	openx := NewOpenxSyncer("localhost")
	syncInfo := openx.GetUsersyncInfo()
	if syncInfo.URL != "https://rtb.openx.net/sync/prebid?r=localhost%2Fsetuid%3Fbidder%3Dopenx%26uid%3D%24%7BUID%7D" {
		t.Fatalf("should have matched")
	}
	if syncInfo.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if syncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}
	if openx.GDPRVendorID() != 69 {
		t.Errorf("Wrong OpenX GDPR VendorID. Got %d", openx.GDPRVendorID())
	}
}
