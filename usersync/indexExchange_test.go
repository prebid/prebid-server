package usersync

import (
	"testing"
)

func TestIndexSyncer(t *testing.T) {
	info := NewIndexSyncer("//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=localhost%2Fsetuid%3Fbidder%3DindexExchange%26uid%3D").GetUsersyncInfo()
	if info.URL != "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=localhost%2Fsetuid%3Fbidder%3DindexExchange%26uid%3D" {
		t.Fatalf("should have matched")
	}
	if info.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
