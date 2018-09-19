package beachfront

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/internal/testutil"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestBeachfrontSyncer(t *testing.T) {
	syncer := NewBeachfrontSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderBeachfront): {
			UserSyncURL: "localhost",
		},
	}})
	u := testutil.UsersyncTest(t, syncer, syncer.GetUsersyncInfo("0", ""))
	u.Assert(
		"localhost",
		"redirect",
		0,
		false,
	)
}
