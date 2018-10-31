package audienceNetwork

import (
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

func NewFacebookSyncer(cfg *config.Configuration) usersync.Usersyncer {
	syncURL := cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderFacebook))].UserSyncURL
	return adapters.NewSyncer("audienceNetwork", 0, adapters.ResolveMacros(syncURL), adapters.SyncTypeRedirect)
}
