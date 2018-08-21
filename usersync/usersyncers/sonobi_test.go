package usersyncers

import (
	"fmt"
	"testing"
)

func TestSonobiSyncer(t *testing.T) {
	url := "https://some.prebid-server.com"
	expectedUrl := "http://sync.go.sonobi.com/us.gif?loc=https%3A%2F%2Fsome.prebid-server.com%2Fsetuid%3Fbidder%3Dsonobi%26uid%3D%24UID"
	syncer := NewSonobiSyncer(url)
	info := syncer.GetUsersyncInfo()
	fmt.Println(info.URL)
	fmt.Println(expectedUrl)
	if info.URL != expectedUrl {
		t.Fatalf("should have matched")
	}
	if info.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}

	if syncer.FamilyName() != "sonobi" {
		t.Errorf("FamilyName '%s' != 'rubicon'", syncer.FamilyName())
	}
}
