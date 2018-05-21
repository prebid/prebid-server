package usersyncers

import (
	"testing"
)

func TestOathSyncer(t *testing.T) {
	url := "http://east-bid.ybp.yahoo.com/sync/appnexuspbs?url=localhost%2Fsetuid%3Fbidder%3Doath%26uid%3D%24%7BUID%7D"

	syncInfo := NewOathSyncer("http://east-bid.ybp.yahoo.com/sync/appnexuspbs?url=%s", "localhost").GetUsersyncInfo()

	if syncInfo.URL != url {
		t.Fatalf("should have matched")
	}
	if syncInfo.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if syncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
