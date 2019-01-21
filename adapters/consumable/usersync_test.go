package consumable

import (
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/spf13/viper"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
)

func TestConsumableSyncer(t *testing.T) {

	// Grab default config
	v := viper.New()
	config.SetupViper(v, "")
	cfg, err := config.New(v)
	if err != nil {
		t.Error(err.Error())
	}
	syncer := NewConsumableSyncer(cfg)

	u := syncer.GetUsersyncInfo("0", "")
	assert.Equal(t, "//e.serverbid.com/udb/9969/match?redir=http%3A%2F%2Flocalhost%3A8000%2Fsetuid%3Fbidder%3Dconsumable%26gdpr%3D0%26gdpr_consent%3D%26uid%3D", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(65535), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}

func TestConsumableSyncerWithConfigOverride(t *testing.T) {
	syncer := NewConsumableSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderConsumable): {
			UserSyncURL: "//sync.hyperbid.com/udb/9969/match?redir=",
		},
	}})

	u := syncer.GetUsersyncInfo("1", "abcdef012344556")
	assert.Equal(t, "//sync.hyperbid.com/udb/9969/match?redir=localhost%2Fsetuid%3Fbidder%3Dconsumable%26gdpr%3D1%26gdpr_consent%3Dabcdef012344556%26uid%3D", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(65535), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
