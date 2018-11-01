package ttx

import (
	"net/url"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

func New33AcrossSyncer(cfg *config.Configuration) usersync.Usersyncer {
	externalURL := strings.TrimRight(cfg.ExternalURL, "/")
	adapterConfig := cfg.Adapters[string(openrtb_ext.Bidder33Across)]
	redirectURL := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dttx%26uid%3D33XUSERID33X"
	syncerURL := adapterConfig.UserSyncURL + "/?ri=" + adapterConfig.PartnerId + "&ru=" + redirectURL

	if adapterConfig.PartnerId == "" {
		syncerURL = "/"
	}

	return adapters.NewSyncer("ttx", 58, adapters.ResolveMacros(syncerURL), adapters.SyncTypeRedirect)
}
