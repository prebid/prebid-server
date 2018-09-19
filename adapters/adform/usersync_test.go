package adform

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/internal/testutil"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestAdformSyncer(t *testing.T) {
	syncer := NewAdformSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderAdform): {
			UserSyncURL: "//cm.adform.net?return_url=",
		},
	}})
	u := testutil.UsersyncTest(t, syncer, syncer.GetUsersyncInfo("1", "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw"))
	u.Assert(
		"//cm.adform.net?return_url=localhost%2Fsetuid%3Fbidder%3Dadform%26gdpr%3D1%26gdpr_consent%3DBONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw%26uid%3D%24UID",
		"redirect",
		50,
		false,
	)
}
