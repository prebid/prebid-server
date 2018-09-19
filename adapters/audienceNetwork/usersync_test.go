package audienceNetwork

import (
	"strings"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/internal/testutil"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestFacebookSyncer(t *testing.T) {
	syncer := NewFacebookSyncer(&config.Configuration{Adapters: map[string]config.Adapter{
		strings.ToLower(string(openrtb_ext.BidderFacebook)): {
			UserSyncURL: "https://www.facebook.com/audiencenetwork/idsync/?partner=partnerId&callback=localhost%2Fsetuid%3Fbidder%3DaudienceNetwork%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24UID",
		},
	}})
	u := testutil.UsersyncTest(t, syncer, syncer.GetUsersyncInfo("", ""))
	u.Assert(
		"https://www.facebook.com/audiencenetwork/idsync/?partner=partnerId&callback=localhost%2Fsetuid%3Fbidder%3DaudienceNetwork%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24UID",
		"redirect",
		0,
		false,
	)
}
