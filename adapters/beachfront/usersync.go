package beachfront

import (
	"fmt"

	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/usersync"
)

func NewBeachfrontSyncer(cfg *config.Configuration) usersync.Usersyncer {
	b := string(openrtb_ext.BidderBeachfront)
	usersyncURL := cfg.Adapters[b].UserSyncURL
	platformID := cfg.Adapters[b].PlatformID
	url := fmt.Sprintf("%s%s", usersyncURL, platformID)
	return adapters.NewSyncer("beachfront", 0, adapters.ResolveMacros(url), adapters.SyncTypeRedirect)
}
