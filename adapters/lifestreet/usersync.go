package lifestreet

import (
	"net/url"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/usersync"
)

func NewLifestreetSyncer(cfg *config.Configuration) usersync.Usersyncer {
	redirectURI := url.QueryEscape(cfg.ExternalURL) + "%2Fsetuid%3Fbidder%3Dlifestreet%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24%24visitor_cookie%24%24"
	usersyncURL := "//ads.lfstmedia.com/idsync/137062?synced=1&ttl=1s&rurl="
	return adapters.NewSyncer("lifestreet", 67, adapters.ResolveMacros(usersyncURL+redirectURI), adapters.SyncTypeRedirect)
}
