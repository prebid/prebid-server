package usersync

import (
	"testing"
)

func TestEPlanningSyncer(t *testing.T) {

	url := "http://sync.e-planning.net/um?uidlocalhost%2Fsetuid%3Fbidder%3Deplanning%26uid%3D%24UID"

	info := NewEPlanningSyncer("http://sync.e-planning.net/um?uid", "localhost").GetUsersyncInfo()
	if info.URL != url {
		t.Fatalf("User Sync Info URL '%s' doesn't match '%s'", info.URL, url)
	}
	if info.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
}
