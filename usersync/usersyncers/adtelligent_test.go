package usersyncers

import (
	"strings"
	"testing"
)

func TestAdtelligentSyncer(t *testing.T) {
	an := NewAdtelligentSyncer("localhost")
	syncInfo := an.GetUsersyncInfo("", "")

	csyncPath := "csync?t=p&ep=0&redir=localhost%2Fsetuid%3Fbidder%3Dadtelligent%26uid%3D%7Buid%7D"
	if !strings.Contains(syncInfo.URL, csyncPath) {
		t.Fatalf("bad url suffix. Expected %s got %s", csyncPath, syncInfo.URL)
	}
	assertStringsMatch(t, "redirect", syncInfo.Type)
	if syncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
