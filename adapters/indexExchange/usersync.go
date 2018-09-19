package indexExchange

import (
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

func NewIndexSyncer(cfg *config.Configuration) usersync.Usersyncer {
	userSyncURL := cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderIndex))].UserSyncURL
	return adapters.NewSyncer("indexExchange", 10, adapters.ResolveMacros(userSyncURL), adapters.SyncTypeRedirect)
}
