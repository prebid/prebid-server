package pubmatic

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/internal/testutil"
)

func TestPubmaticSyncer(t *testing.T) {
	syncer := NewPubmaticSyncer(&config.Configuration{ExternalURL: "localhost"})
	u := testutil.UsersyncTest(t, syncer, syncer.GetUsersyncInfo("1", "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw"))
	u.Assert(
		"//ads.pubmatic.com/AdServer/js/user_sync.html?predirect=localhost%2Fsetuid%3Fbidder%3Dpubmatic%26gdpr%3D1%26gdpr_consent%3DBONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw%26uid%3D",
		"iframe",
		76,
		false,
	)
}
