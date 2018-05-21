package usersyncers

import (
	"testing"
)

func TestEPlanningSyncer(t *testing.T) {

	url := "http://sync.e-planning.net/um?uidlocalhost%2Fsetuid%3Fbidder%3Deplanning%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24UID"

	info := NewEPlanningSyncer("http://sync.e-planning.net/um?uid", "localhost").GetUsersyncInfo("", "")
	assertStringsMatch(t, url, info.URL)
	assertStringsMatch(t, "redirect", info.Type)
}
