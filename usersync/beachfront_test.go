package usersync

import "testing"

func TestBeachfrontSyncer(t *testing.T) {
	an := NewBeachfrontSyncer("localhost", "localhost")
	syncInfo := an.GetUsersyncInfo()

	if syncInfo.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if syncInfo.SupportCORS != true {
		t.Fatalf("should have been true")
	}
}
