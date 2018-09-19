package appnexus

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/internal/testutil"
)

func TestAppNexusSyncer(t *testing.T) {
	syncer := NewAppnexusSyncer(&config.Configuration{ExternalURL: "https://prebid.adnxs.com/pbs/v1"})
	u := testutil.UsersyncTest(t, syncer, syncer.GetUsersyncInfo("", ""))
	u.Assert(
		"//ib.adnxs.com/getuid?https%3A%2F%2Fprebid.adnxs.com%2Fpbs%2Fv1%2Fsetuid%3Fbidder%3Dadnxs%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24UID",
		"redirect",
		32,
		false,
	)
}
