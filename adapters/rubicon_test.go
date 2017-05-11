package adapters

import (
	"testing"
)

func TestRubiconUserSyncInfo(t *testing.T) {

	an := NewRubiconAdapter(DefaultHTTPAdapterConfig, "uri", "xuser", "xpass", "localhost")
	if an.usersyncInfo.URL != "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid" {
		t.Fatalf("should have matched")
	}
	if an.usersyncInfo.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if an.usersyncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}

}
