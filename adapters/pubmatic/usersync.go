package pubmatic

import (
	"net/url"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/usersync"
)

func NewPubmaticSyncer(cfg *config.Configuration) usersync.Usersyncer {
	redirectURI := url.QueryEscape(cfg.ExternalURL) + "%2Fsetuid%3Fbidder%3Dpubmatic%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D"
	usersyncURL := "//ads.pubmatic.com/AdServer/js/user_sync.html?predirect="
	return adapters.NewSyncer("pubmatic", 76, adapters.ResolveMacros(usersyncURL+redirectURI), adapters.SyncTypeIframe)
}
