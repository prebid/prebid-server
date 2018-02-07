package usersync

import (
	"testing"
)

func TestAppNexusSyncer(t *testing.T) {
	an := NewAppnexusSyncer("localhost")
	syncInfo := an.GetUsersyncInfo()
	if syncInfo.URL != "//ib.adnxs.com/getuid?localhost%2Fsetuid%3Fbidder%3Dadnxs%26uid%3D%24UID" {
		t.Fatalf("should have matched")
	}
	if syncInfo.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if syncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
