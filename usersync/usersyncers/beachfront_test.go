package usersyncers

import "testing"

func TestBeachfrontSyncer(t *testing.T) {
	an := NewBeachfrontSyncer("localhost", "localhost")
	syncInfo := an.GetUsersyncInfo("0", "")

	if syncInfo.Type != "redirect" {
		t.Fatalf("Type should be redirect")
	}
	if syncInfo.SupportCORS != false {
		t.Fatalf("SupportCORS should have been fasle")
	}
}
