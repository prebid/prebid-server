package openx

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/internal/testutil"
)

func TestOpenxSyncer(t *testing.T) {
	syncer := NewOpenxSyncer(&config.Configuration{ExternalURL: "localhost"})
	u := testutil.UsersyncTest(t, syncer, syncer.GetUsersyncInfo("", ""))
	u.Assert(
		"https://rtb.openx.net/sync/prebid?r=localhost%2Fsetuid%3Fbidder%3Dopenx%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24%7BUID%7D",
		"redirect",
		69,
		false,
	)
}
