package conversant

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestConversantSyncer(t *testing.T) {
	syncer := NewConversantSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderConversant): {
			UserSyncURL: "usersync?rurl=",
		},
	}})
	u := syncer.GetUsersyncInfo("0", "")
	assert.Equal(t, "usersync?rurl=localhost%2Fsetuid%3Fbidder%3Dconversant%26gdpr%3D0%26gdpr_consent%3D%26uid%3D", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(24), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
