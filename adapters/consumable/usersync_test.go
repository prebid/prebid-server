package consumable

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
)

func TestConsumableSyncer(t *testing.T) {
	syncer := NewConsumableSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		// If external config were used we'd configure it here
		//string(openrtb_ext.BidderConsumable): {
		//	UserSyncURL: "//sync.serverbid.com/ss/0.html?redirect=",
		//},
	}})
	u := syncer.GetUsersyncInfo("", "")
	assert.Equal(t, "//e.serverbid.com/udb/9969/match?redir=localhost%2Fsetuid%3Fbidder%3Dconsumable%26gdpr%3D%26gdpr_consent%3D%26uid%3D", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(65535), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
