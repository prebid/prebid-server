package usersyncers

import (
	"testing"
)

func TestAdformSyncer(t *testing.T) {
	an := NewAdformSyncer("//cm.adform.net?return_url=", "localhost")
	syncInfo := an.GetUsersyncInfo("", "")
	assertStringsMatch(t, "//cm.adform.net?return_url=localhost%2Fsetuid%3Fbidder%3Dadform%26uid%3D%24UID", syncInfo.URL)
	assertStringsMatch(t, "redirect", syncInfo.Type)
	if syncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
