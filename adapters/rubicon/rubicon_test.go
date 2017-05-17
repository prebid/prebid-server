package rubicon

import (
	"testing"
)

func TestRubiconUserSyncInfo(t *testing.T) {
	url := "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid"

	an := NewRubiconAdapter(DefaultHTTPAdapterConfig, "uri", "xuser", "xpass", url)
	if an.usersyncInfo.URL != url {
		t.Fatalf("should have matched")
	}
	if an.usersyncInfo.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if an.usersyncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}

}
