package usersyncers

import (
	"testing"
)

func TestSomoaudienceSyncer(t *testing.T) {
	somo := NewSomoaudienceSyncer("localhost")
	syncInfo := somo.GetUsersyncInfo("", "")
	if syncInfo.URL != "//publisher-east.mobileadtrading.com/usersync?ru=localhost%2Fsetuid%3Fbidder%3Dsomoaudience%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24%7BUID%7D" {
		t.Fatalf("should have matched")
	}
	if syncInfo.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if syncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
