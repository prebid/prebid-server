package usersyncers

import (
	"testing"
)

func Test33AcrossSyncer(t *testing.T) {
	ttx := New33AcrossSyncer("http://localhost", "https://ssc-cms.33across.com/ps", "123")
	syncInfo := ttx.GetUsersyncInfo("", "")
	assertStringsMatch(t, "https://ssc-cms.33across.com/ps/?ri=123&ru=http%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dttx%26uid%3D33XUSERID33X", syncInfo.URL)
	assertStringsMatch(t, "redirect", syncInfo.Type)
	if syncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
