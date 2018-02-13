package usersync

import (
	"testing"
)

func TestAdformSyncer(t *testing.T) {
	an := NewAdformSyncer("//cm.adform.net?return_url=", "localhost")
	syncInfo := an.GetUsersyncInfo()
	if syncInfo.URL != "//cm.adform.net?return_url=localhost%2Fsetuid%3Fbidder%3Dadform%26uid%3D%24UID" {
		t.Fatalf("should have matched")
	}
	if syncInfo.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if syncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
