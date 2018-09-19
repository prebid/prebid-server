package conversant

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/internal/testutil"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestConversantSyncer(t *testing.T) {
	syncer := NewConversantSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderConversant): {
			UserSyncURL: "usersync?rurl=",
		},
	}})
	u := testutil.UsersyncTest(t, syncer, syncer.GetUsersyncInfo("0", ""))
	u.Assert(
		"usersync?rurl=localhost%2Fsetuid%3Fbidder%3Dconversant%26gdpr%3D0%26gdpr_consent%3D%26uid%3D",
		"redirect",
		24,
		false,
	)
}
