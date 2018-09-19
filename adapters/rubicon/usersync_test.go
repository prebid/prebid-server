package rubicon

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/internal/testutil"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestRubiconSyncer(t *testing.T) {
	syncer := NewRubiconSyncer(&config.Configuration{Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderRubicon): {
			UserSyncURL: "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid&gdpr={{gdpr}}&gdpr_consent={{gdpr_consent}}",
		},
	}})
	u := testutil.UsersyncTest(t, syncer, syncer.GetUsersyncInfo("0", ""))
	u.Assert(
		"https://pixel.rubiconproject.com/exchange/sync.php?p=prebid&gdpr=0&gdpr_consent=",
		"redirect",
		52,
		false,
	)
	u.AssertFamilyName("rubicon")
}
