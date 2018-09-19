package brightroll

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/internal/testutil"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestBrightrollSyncer(t *testing.T) {
	syncer := NewBrightrollSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderBrightroll): {
			UserSyncURL: "http://east-bid.ybp.yahoo.com/sync/appnexuspbs?gdpr={{gdpr}}&euconsent={{gdpr_consent}}&url=",
		},
	}})
	u := testutil.UsersyncTest(t, syncer, syncer.GetUsersyncInfo("", ""))
	u.Assert(
		"http://east-bid.ybp.yahoo.com/sync/appnexuspbs?gdpr=&euconsent=&url=localhost%2Fsetuid%3Fbidder%3Dbrightroll%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24%7BUID%7D",
		"redirect",
		25,
		false,
	)
}
