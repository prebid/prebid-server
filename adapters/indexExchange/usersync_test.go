package indexExchange

import (
	"strings"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/internal/testutil"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestIndexSyncer(t *testing.T) {
	syncer := NewIndexSyncer(&config.Configuration{Adapters: map[string]config.Adapter{
		strings.ToLower(string(openrtb_ext.BidderIndex)): {
			UserSyncURL: "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=localhost%2Fsetuid%3Fbidder%3DindexExchange%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D",
		},
	}})
	u := testutil.UsersyncTest(t, syncer, syncer.GetUsersyncInfo("", ""))
	u.Assert(
		"//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=localhost%2Fsetuid%3Fbidder%3DindexExchange%26gdpr%3D%26gdpr_consent%3D%26uid%3D",
		"redirect",
		10,
		false,
	)
}
