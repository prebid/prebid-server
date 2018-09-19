package pulsepoint

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/internal/testutil"
)

func TestPulsepointSyncer(t *testing.T) {
	syncer := NewPulsepointSyncer(&config.Configuration{ExternalURL: "http://localhost"})
	u := testutil.UsersyncTest(t, syncer, syncer.GetUsersyncInfo("", ""))
	u.Assert(
		"//bh.contextweb.com/rtset?pid=561205&ev=1&rurl=http%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dpulsepoint%26gdpr%3D%26gdpr_consent%3D%26uid%3D%25%25VGUID%25%25",
		"redirect",
		81,
		false,
	)
}
