package usersync

import (
	"testing"
)

func TestPubmaticSyncer(t *testing.T) {
	info := NewPubmaticSyncer("localhost").GetUsersyncInfo()
	if info.URL != "//ads.pubmatic.com/AdServer/js/user_sync.html?predirect=localhost%2Fsetuid%3Fbidder%3Dpubmatic%26uid%3D" {
		t.Fatalf("should have matched")
	}
	if info.Type != "iframe" {
		t.Fatalf("should be iframe")
	}
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
