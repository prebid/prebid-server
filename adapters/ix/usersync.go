package ix

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

func NewIxSyncer(cfg *config.Configuration) usersync.Usersyncer {
	usersyncURL := cfg.Adapters[string(openrtb_ext.BidderIx)].UserSyncURL
	return adapters.NewSyncer("ix", 10, adapters.ResolveMacros(usersyncURL), adapters.SyncTypeRedirect)
}
