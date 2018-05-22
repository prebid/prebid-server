package usersyncers

import (
	"testing"
)

func TestAdformSyncer(t *testing.T) {
	an := NewAdformSyncer("//cm.adform.net?return_url=", "localhost")
	syncInfo := an.GetUsersyncInfo("1", "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw")
	url := "//cm.adform.net?return_url=localhost%2Fsetuid%3Fbidder%3Dadform%26gdpr%3D1%26gdpr_consent%3DBONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw%26uid%3D%24UID"
	assertStringsMatch(t, url, syncInfo.URL)
	assertStringsMatch(t, "redirect", syncInfo.Type)
	if syncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
