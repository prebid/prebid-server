package eplanning

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/internal/testutil"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestEPlanningSyncer(t *testing.T) {
	syncer := NewEPlanningSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderEPlanning): {
			UserSyncURL: "http://sync.e-planning.net/um?uid",
		},
	}})
	u := testutil.UsersyncTest(t, syncer, syncer.GetUsersyncInfo("", ""))
	u.Assert(
		"http://sync.e-planning.net/um?uidlocalhost%2Fsetuid%3Fbidder%3Deplanning%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24UID",
		"redirect",
		0,
		false,
	)
}
