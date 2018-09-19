package adtelligent

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/internal/testutil"
)

func TestAdtelligentSyncer(t *testing.T) {
	syncer := NewAdtelligentSyncer(&config.Configuration{ExternalURL: "localhost"})
	u := testutil.UsersyncTest(t, syncer, syncer.GetUsersyncInfo("0", ""))
	u.Assert(
		"//sync.adtelligent.com/csync?t=p&ep=0&redir=localhost%2Fsetuid%3Fbidder%3Dadtelligent%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%7Buid%7D",
		"redirect",
		0,
		false,
	)
}
