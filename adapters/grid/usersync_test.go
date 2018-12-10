package grid

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestGridSyncer(t *testing.T) {
	syncer := NewGridSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderGrid): {
			UserSyncURL: "//not_localhost/sync",
		},
	}})
	u := syncer.GetUsersyncInfo("0", "")
	assert.Equal(t, "//not_localhost/synclocalhost%2Fsetuid%3Fbidder%3Dgrid%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24UID", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(0), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
