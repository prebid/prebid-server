package adapters_test

import (
	"testing"

	"github.com/prebid/prebid-server/adapters"
)

func TestRubiconUserSyncInfo(t *testing.T) {
	url := "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid"

	an := adapters.NewRubiconAdapter(adapters.DefaultHTTPAdapterConfig, "uri", "xuser", "xpass", url)
	if an.GetUsersyncInfo().URL != url {
		t.Fatalf("should have matched")
	}
	if an.GetUsersyncInfo().Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if an.GetUsersyncInfo().SupportCORS != false {
		t.Fatalf("should have been false")
	}

}
