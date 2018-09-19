package appnexus

import (
	"net/url"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/usersync"
)

func NewAppnexusSyncer(cfg *config.Configuration) usersync.Usersyncer {
	redirectURI := url.QueryEscape(cfg.ExternalURL) + "%2Fsetuid%3Fbidder%3Dadnxs%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24UID"
	usersyncURL := "//ib.adnxs.com/getuid?"
	return adapters.NewSyncer("adnxs", 32, adapters.ResolveMacros(usersyncURL+redirectURI), adapters.SyncTypeRedirect)
}
