package lifestreet

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/internal/testutil"
)

func TestLifestreetSyncer(t *testing.T) {
	syncer := NewLifestreetSyncer(&config.Configuration{ExternalURL: "localhost"})
	u := testutil.UsersyncTest(t, syncer, syncer.GetUsersyncInfo("0", ""))
	u.Assert(
		"//ads.lfstmedia.com/idsync/137062?synced=1&ttl=1s&rurl=localhost%2Fsetuid%3Fbidder%3Dlifestreet%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24%24visitor_cookie%24%24",
		"redirect",
		67,
		false,
	)
}
