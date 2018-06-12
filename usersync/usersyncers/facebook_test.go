package usersyncers

import (
	"testing"
)

func TestFacebookSyncer(t *testing.T) {
	url := "https://www.facebook.com/audiencenetwork/idsync/?partner=partnerId&callback=localhost%2Fsetuid%3Fbidder%3DaudienceNetwork%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24UID"
	expected := "https://www.facebook.com/audiencenetwork/idsync/?partner=partnerId&callback=localhost%2Fsetuid%3Fbidder%3DaudienceNetwork%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24UID"
	// %26gdpr%3D%26gdpr_consent%3D

	info := NewFacebookSyncer(url).GetUsersyncInfo("", "")
	assertStringsMatch(t, expected, info.URL)
	assertStringsMatch(t, "redirect", info.Type)
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
