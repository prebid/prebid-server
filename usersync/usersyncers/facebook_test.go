package usersyncers

import (
	"testing"
)

func TestFacebookSyncer(t *testing.T) {
	url := "https://www.facebook.com/audiencenetwork/idsync/?partner=partnerId&callback=localhost%2Fsetuid%3Fbidder%3DaudienceNetwork%26uid%3D%24UID"

	info := NewFacebookSyncer(url).GetUsersyncInfo("", "")
	assertStringsMatch(t, url, info.URL)
	assertStringsMatch(t, "redirect", info.Type)
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
