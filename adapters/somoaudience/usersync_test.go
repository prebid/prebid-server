package somoaudience

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/internal/testutil"
)

func TestSomoaudienceSyncer(t *testing.T) {
	syncer := NewSomoaudienceSyncer(&config.Configuration{ExternalURL: "localhost"})
	u := testutil.UsersyncTest(t, syncer, syncer.GetUsersyncInfo("", ""))
	u.Assert(
		"//publisher-east.mobileadtrading.com/usersync?ru=localhost%2Fsetuid%3Fbidder%3Dsomoaudience%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24%7BUID%7D",
		"redirect",
		341,
		false,
	)
}
